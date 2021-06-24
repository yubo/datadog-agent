package registration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/DataDog/datadog-agent/pkg/util/log"
)

const (
	headerContentType  string = "Content-Type"
	routeSubscribeLogs string = "/2020-08-15/logs"
)

//requestTimeout     time.Duration = 5 * time.Second

// SubscribeLogs subscribes to the logs collection on the platform.
// We send a request to AWS to subscribe for logs, indicating on which port we
// are opening an HTTP server, to receive logs from AWS.
// When we are receiving logs on this HTTP server, we're pushing them in a channel
// tailed by the Logs Agent pipeline, these logs then go through the regular
// Logs Agent pipeline to finally be sent on the intake when we receive a FLUSH
// call from the Lambda function / client.
// logsType contains the type of logs for which we are subscribing, possible
// value: platform, extension and function.
func SubscribeLogs(id ID, url string, logsType []string, timeout time.Duration) error {
	log.Debug("Subscribing to Logs for types:", logsType)

	jsonBytes, err := buildLogRegistrationPayload(url, logsType, 1000, 262144, 1000)
	if err != nil {
		return fmt.Errorf("SubscribeLogs: can't marshal subscribe JSON %v", err)
	}

	request, err := buildLogRegistrationRequest(url, HeaderExtID, headerContentType, id, jsonBytes)
	if err != nil {
		return fmt.Errorf("SubscribeLogs: can't create the PUT request: %v", err)
	}

	response, err := sendLogRegistrationRequest(&http.Client{
		Transport: &http.Transport{IdleConnTimeout: timeout},
		Timeout:   timeout,
	}, request)
	if err != nil {
		return fmt.Errorf("SubscribeLogs: while PUT subscribe request: %s", err)
	}

	if !isValidHttpCode(response.StatusCode) {
		return fmt.Errorf("SubscribeLogs: received an HTTP %s", response.Status)
	}

	return nil
}

func buildLogRegistrationPayload(httpAddr string, logsType []string, timeoutMs int, maxBytes int, maxItems int) ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"destination": map[string]string{
			"URI":      httpAddr,
			"protocol": "HTTP",
		},
		"types": logsType,
		"buffering": map[string]int{
			"timeoutMs": timeoutMs,
			"maxBytes":  maxBytes,
			"maxItems":  maxItems,
		},
	})
}

func buildLogRegistrationRequest(url string, headerExtID string, headerContentType string, id ID, payload []byte) (*http.Request, error) {
	request, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}
	request.Header.Set(headerExtID, id.String())
	request.Header.Set(headerContentType, "application/json")
	return request, nil
}

func sendLogRegistrationRequest(httpClient HttpClient, request *http.Request) (*http.Response, error) {
	return httpClient.Do(request)
}

func isValidHttpCode(statusCode int) bool {
	return statusCode < 300
}
