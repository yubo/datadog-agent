package registration

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const registerLogsTimeout = 10 * time.Millisecond

func TestBuildLogRegistrationPayload(t *testing.T) {
	payload := BuildLogRegistrationPayload("myUri", []string{"logType1", "logType2"}, 10, 100, 1000)
	assert.Equal(t, "HTTP", payload.Destination.Protocol)
	assert.Equal(t, "myUri", payload.Destination.URI)
	assert.Equal(t, 10, payload.Buffering.TimeoutMs)
	assert.Equal(t, 100, payload.Buffering.MaxBytes)
	assert.Equal(t, 1000, payload.Buffering.MaxItems)
	assert.Equal(t, []string{"logType1", "logType2"}, payload.Types)
}

func TestBuildLogRegistrationRequestSuccess(t *testing.T) {
	request, err := buildLogRegistrationRequest("myUrl", "X-Extension", "Content-Type", "myId", []byte("test"))
	assert.Nil(t, err)
	assert.Equal(t, http.MethodPut, request.Method)
	assert.Equal(t, "myUrl", request.URL.Path)
	assert.NotNil(t, request.Body)
	assert.Equal(t, "myId", request.Header["X-Extension"][0])
	assert.Equal(t, "application/json", request.Header["Content-Type"][0])
}

func TestBuildLogRegistrationRequestError(t *testing.T) {
	request, err := buildLogRegistrationRequest(":invalid:", "X-Extension", "Content-Type", "myId", []byte("test"))
	assert.NotNil(t, err)
	assert.Nil(t, request)
}

func TestIsValidHttpCodeSuccess(t *testing.T) {
	assert.True(t, isValidHttpCode(200))
	assert.True(t, isValidHttpCode(202))
	assert.True(t, isValidHttpCode(204))
}

func TestIsValidHttpCodeError(t *testing.T) {
	assert.False(t, isValidHttpCode(300))
	assert.False(t, isValidHttpCode(404))
	assert.False(t, isValidHttpCode(400))
}

func TestSendLogRegistrationRequestFailure(t *testing.T) {
	response, err := sendLogRegistrationRequest(&http.Client{}, &http.Request{})
	assert.Nil(t, response)
	assert.NotNil(t, err)
}

func TestSendLogRegistrationRequestSuccess(t *testing.T) {
	response, err := sendLogRegistrationRequest(&ClientMock{}, &http.Request{})
	assert.Nil(t, err)
	assert.NotNil(t, response)
}

func TestSubscribeLogsSuccess(t *testing.T) {
	payload := BuildLogRegistrationPayload("myUri", []string{"logType1", "logType2"}, 10, 100, 1000)
	//fake the register route
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer ts.Close()

	err := SubscribeLogs("myId", ts.URL, registerLogsTimeout, payload)
	assert.Nil(t, err)
}

func TestSubscribeLogsTimeout(t *testing.T) {
	payload := BuildLogRegistrationPayload("myUri", []string{"logType1", "logType2"}, 10, 100, 1000)
	//fake the register route
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// timeout
		time.Sleep(registerLogsTimeout + 10*time.Millisecond)
		w.WriteHeader(200)
	}))
	defer ts.Close()

	err := SubscribeLogs("myId", ts.URL, registerLogsTimeout, payload)
	assert.NotNil(t, err)
}

func TestSubscribeLogsInvalidHttpCode(t *testing.T) {
	payload := BuildLogRegistrationPayload("myUri", []string{"logType1", "logType2"}, 10, 100, 1000)
	//fake the register route
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// invalid code
		w.WriteHeader(500)
	}))
	defer ts.Close()

	err := SubscribeLogs("myId", ts.URL, registerLogsTimeout, payload)
	assert.NotNil(t, err)
}

func TestSubscribeLogsInvalidUrl(t *testing.T) {
	payload := BuildLogRegistrationPayload("myUri", []string{"logType1", "logType2"}, 10, 100, 1000)
	err := SubscribeLogs("myId", ":invalid:", registerLogsTimeout, payload)
	assert.NotNil(t, err)
}

type ImpossibleToMarshall struct{}

func (p *ImpossibleToMarshall) MarshalJSON() ([]byte, error) {
	return nil, fmt.Errorf("should fail")
}

func TestSubscribeLogsInvalidPayloadObject(t *testing.T) {
	payload := &ImpossibleToMarshall{}
	err := SubscribeLogs("myId", ":invalid:", registerLogsTimeout, payload)
	assert.NotNil(t, err)
}
