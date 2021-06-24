package serverless

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"time"
)

const (
	extensionName        = "datadog-agent"
	headerExtName        = "Lambda-Extension-Name"
	headerExtID   string = "Lambda-Extension-Identifier"
)

// Register registers the serverless daemon and subscribe to INVOKE and SHUTDOWN messages.
// Returns either (the serverless ID assigned by the serverless daemon + the api key as read from
// the environment) or an error.
func Register(url string) (ID, error) {
	var err error

	payload := createRegistrationPayload()

	request := buildRegisterRequest(headerExtName, extensionName, url, payload)
	if request == nil {
		return "", fmt.Errorf("Register: can't create the POST register request: %v", err)
	}

	response := sendRequest(request)
	if response == nil {
		return "", fmt.Errorf("Register: error while POST register route: %v", err)
	}

	if !isAValidResponse(response) {
		return "", fmt.Errorf("Register: didn't receive an HTTP 200")
	}

	id := extractId(response)
	if len(id) == 0 {
		return "", fmt.Errorf("Register: didn't receive an identifier")
	}

	return ID(id), nil
}

func buildURL(route string) string {
	prefix := os.Getenv("AWS_LAMBDA_RUNTIME_API")
	if len(prefix) == 0 {
		return fmt.Sprintf("http://localhost:9001%s", route)
	}
	return fmt.Sprintf("http://%s%s", prefix, route)
}

func createRegistrationPayload() *bytes.Buffer {
	payload := bytes.NewBuffer(nil)
	payload.Write([]byte(`{"events":["INVOKE", "SHUTDOWN"]}`))
	return payload
}

func extractId(response *http.Response) string {
	return response.Header.Get(headerExtID)
}

func isAValidResponse(response *http.Response) bool {
	return response.StatusCode == 200
}

func buildRegisterRequest(headerExtensionName string, extensionName string, url string, payload *bytes.Buffer) *http.Request {
	request, err := http.NewRequest(http.MethodPost, url, payload)
	if err != nil {
		return nil
	}
	request.Header.Set(headerExtName, extensionName)
	return request
}

func sendRequest(request *http.Request) *http.Response {
	client := &http.Client{Timeout: 5 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		return nil
	}
	return response
}
