package metadata

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	stdHttp "net/http"
	"net/url"
	"path"
	"regexp"
	"strings"
	"time"

	coreConfig "github.com/DataDog/datadog-agent/pkg/config"
	"github.com/DataDog/datadog-agent/pkg/logs/config"
	"github.com/DataDog/datadog-agent/pkg/metadata/host"
	"github.com/DataDog/datadog-agent/pkg/util"
	httputils "github.com/DataDog/datadog-agent/pkg/util/http"
	"github.com/DataDog/datadog-agent/pkg/util/log"
	"github.com/DataDog/datadog-agent/pkg/version"
)

const (
	// MetaPollInterval Polling interval in Seconds
	MetaPollInterval time.Duration = 2 * time.Second
	// MetaEndpoint API endpoint
	MetaEndpoint string = "api/v1/tags/hosts/"
)

//Ready waits for metadata to be available
func Ready(ctx context.Context, endpoint string, timeout int) error {
	timer := time.NewTimer(time.Duration(timeout) * time.Second)
	ticker := time.NewTicker(MetaPollInterval)

	var api string
	re := regexp.MustCompile(`datadoghq.(com|eu){1}$`)
	if re.MatchString(endpoint) {
		api = path.Join(fmt.Sprintf("api.%s", re.FindString(endpoint)), MetaEndpoint)
	} else {
		message := fmt.Sprintf("unsupported target domain: %s", endpoint)
		return errors.New(message)
	}

	tags := host.GetHostTags(true).System
	expectedTagKeys := coreConfig.Datadog.GetStringSlice("expected_external_tags")

	backoff := 0
	for {
		select {
		case <-ctx.Done():
			return errors.New("Metadata tag availability wait canceled")
		case <-timer.C:
			return errors.New("unable to resolve metadata in time")
		case <-ticker.C:
			if backoff > 0 {
				backoff--
				continue
			}
			found, err := pollMeta(api, tags, expectedTagKeys)
			if err != nil {
				log.Infof("There was an issue grabbing the host tags, backing off: %v", err)
				backoff = backoff * 2
			} else if found {
				return nil
			}
			backoff = 0
		}
	}
}

func pollMeta(endpoint string, exactTags []string, tagKeys []string) (bool, error) {
	hostname, err := util.GetHostname()
	if err != nil {
		return false, err
	}

	uri, err := url.Parse(endpoint)
	if err != nil {
		return false, err
	}
	uri.Scheme = "https"

	uri.Path = path.Join(uri.Path, hostname)
	transport := httputils.CreateHTTPTransport()

	// TODO: set a timeout on the client
	client := &stdHttp.Client{
		Transport: transport,
	}

	log.Debugf("Polling for metadata: %v", uri)
	req, err := stdHttp.NewRequest("GET", uri.String(), nil)
	if err != nil {
		return false, err
	}

	req.Header.Set("User-Agent", fmt.Sprintf("datadog-agent/%s", version.AgentVersion))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("DD-API-KEY", coreConfig.Datadog.GetString("api_key"))
	req.Header.Set("DD-APPLICATION-KEY", coreConfig.Datadog.GetString("app_key"))

	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	// Server will respond 200 if the key is valid or 403 if invalid
	if resp.StatusCode == 200 {
		var jsonResponse map[string]interface{}

		json.NewDecoder(resp.Body).Decode(&jsonResponse)
		log.Debugf("metadata response received: %v", jsonResponse)
		_, found := jsonResponse["tags"]
		if !found {
			return false, nil
		}

		tagSet := make(map[string]struct{})
		for _, tag := range jsonResponse["tags"].([]interface{}) {
			tagSet[tag.(string)] = struct{}{}
		}

		for _, tag := range exactTags {
			_, ok := tagSet[tag]
			if !ok {
				return false, nil
			}
		}

		for _, key := range tagKeys {
			match := false
			for tag := range tagSet {
				if strings.HasPrefix(tag, fmt.Sprintf("%s:", key)) {
					match = true
				}
			}
			if !match {
				return false, nil
			}
		}

		return true, nil

	} else if resp.StatusCode == 403 {
		return false, nil
	}

	return false, nil
}
