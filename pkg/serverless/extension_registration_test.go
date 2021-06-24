// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package serverless

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildUrlPrefixEmpty(t *testing.T) {
	builtUrl := buildURL("", "/myPath")
	assert.Equal(t, "http://localhost:9001/myPath", builtUrl)
}

func TestBuildUrlWithPrefix(t *testing.T) {
	builtUrl := buildURL("myPrefix:3000", "/myPath")
	assert.Equal(t, "http://myPrefix:3000/myPath", builtUrl)
}

func TestCreateRegistrationPayload(t *testing.T) {
	payload := createRegistrationPayload()
	assert.Equal(t, "{\"events\":[\"INVOKE\", \"SHUTDOWN\"]}", payload.String())
}

func TestExtractId(t *testing.T) {
	expectedId := "blablabla"
	response := &http.Response{
		Header: map[string][]string{
			headerExtID: []string{expectedId},
		},
	}
	assert.Equal(t, expectedId, extractId(response))
}

func TestIsValidResponseTrue(t *testing.T) {
	response := &http.Response{
		StatusCode: 200,
	}
	assert.True(t, isAValidResponse(response))
}

func TestIsValidResponseFalse(t *testing.T) {
	response := &http.Response{
		StatusCode: 404,
	}
	assert.False(t, isAValidResponse(response))
}

func TestBuildRegisterRequestSuccess(t *testing.T) {
	request, err := buildRegisterRequest("X-Header", "extensionName", "myUrl", bytes.NewBuffer([]byte("blablabla")))
	assert.Nil(t, err)
	assert.Equal(t, http.MethodPost, request.Method)
	assert.Equal(t, "myUrl", request.URL.Path)
	assert.NotNil(t, request.Body)
	assert.Equal(t, "extensionName", request.Header["X-Header"][0])
}

func TestBuildRegisterRequestFailure(t *testing.T) {
	request, err := buildRegisterRequest("X-Header", "extensionName", ":invalid:", bytes.NewBuffer([]byte("blablabla")))
	assert.Nil(t, request)
	assert.NotNil(t, err)
}

func TestFlareHasRightForm(t *testing.T) {
	ts1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts1.Close()

}

type ClientMock struct {
}

func (c *ClientMock) Do(req *http.Request) (*http.Response, error) {
	return &http.Response{}, nil
}

func TestSendRequestFailure(t *testing.T) {
	response, err := sendRequest(&http.Client{}, &http.Request{})
	assert.Nil(t, response)
	assert.NotNil(t, err)
}

func TestSendRequestSuccess(t *testing.T) {
	response, err := sendRequest(&ClientMock{}, &http.Request{})
	assert.Nil(t, err)
	assert.NotNil(t, response)
}

func TestRegisterFailure(t *testing.T) {
	func Register(prefix string, url string) (ID, error) {
		payload := createRegistrationPayload()
	
		request, err := buildRegisterRequest(headerExtName, extensionName, buildURL(prefix, url), payload)
		if err != nil {
			return "", fmt.Errorf("Register: can't create the POST register request: %v", err)
		}
	
		response, err := sendRequest(&http.Client{Timeout: 5 * time.Second}, request)
		if err != nil {
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
}