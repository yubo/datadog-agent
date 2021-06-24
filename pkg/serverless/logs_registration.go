package serverless

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/DataDog/datadog-agent/pkg/util/log"
)

const (
	headerContentType  string        = "Content-Type"
	routeSubscribeLogs string        = "/2020-08-15/logs"
	requestTimeout     time.Duration = 5 * time.Second
)

// SubscribeLogs subscribes to the logs collection on the platform.
// We send a request to AWS to subscribe for logs, indicating on which port we
// are opening an HTTP server, to receive logs from AWS.
// When we are receiving logs on this HTTP server, we're pushing them in a channel
// tailed by the Logs Agent pipeline, these logs then go through the regular
// Logs Agent pipeline to finally be sent on the intake when we receive a FLUSH
// call from the Lambda function / client.
// logsType contains the type of logs for which we are subscribing, possible
// value: platform, extension and function.
func SubscribeLogs(id ID, prefix string, httpAddr string, logsType []string) error {
	log.Debug("Subscribing to Logs for types:", logsType)

	if !isValidHttpAddr(httpAddr) {
		return fmt.Errorf("SubscribeLogs: wrong http addr provided: %s", httpAddr)
	}

	jsonBytes, err := buildLogRegistrationPayload(httpAddr, logsType)
	if err != nil {
		return fmt.Errorf("SubscribeLogs: can't marshal subscribe JSON %v", err)
	}

	request, err := buildLogRegistrationRequest(prefix, headerExtID, headerContentType, id, jsonBytes)
	if err != nil {
		return fmt.Errorf("SubscribeLogs: can't create the PUT request: %v", err)
	}

	response, err := sendLogRegistrationRequest(request)
	if err != nil {
		return fmt.Errorf("SubscribeLogs: while PUT subscribe request: %s", err)
	}

	if !isValidHttpCode(response.StatusCode) {
		return fmt.Errorf("SubscribeLogs: received an HTTP %s", response.Status)
	}

	return nil
}

func buildLogRegistrationPayload(httpAddr string, logsType []string) ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"destination": map[string]string{
			"URI":      httpAddr,
			"protocol": "HTTP",
		},
		"types": logsType,
		"buffering": map[string]int{ // TODO(remy): these should be better defined
			"timeoutMs": 1000,
			"maxBytes":  262144,
			"maxItems":  1000,
		},
	})
}

func buildLogRegistrationRequest(prefix string, headerExtID string, headerContentType string, id ID, payload []byte) (*http.Request, error) {
	request, err := http.NewRequest(http.MethodPut, buildURL(prefix, routeSubscribeLogs), bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}
	request.Header.Set(headerExtID, id.String())
	request.Header.Set(headerContentType, "application/json")
	return request, nil
}

func sendLogRegistrationRequest(request *http.Request) (*http.Response, error) {
	client := &http.Client{
		Transport: &http.Transport{IdleConnTimeout: requestTimeout},
		Timeout:   requestTimeout,
	}
	return client.Do(request)
}

func isValidHttpCode(statusCode int) bool {
	return statusCode < 300
}

func isValidHttpAddr(httpAddr string) bool {
	_, err := url.ParseRequestURI(httpAddr)
	return err == nil && len(httpAddr) > 0
}
