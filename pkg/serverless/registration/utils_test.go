package registration

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

type ClientMock struct {
}

func (c *ClientMock) Do(req *http.Request) (*http.Response, error) {
	return &http.Response{}, nil
}

func TestBuildUrlPrefixEmpty(t *testing.T) {
	builtUrl := BuildURL("", "/myPath")
	assert.Equal(t, "http://localhost:9001/myPath", builtUrl)
}

func TestBuildUrlWithPrefix(t *testing.T) {
	builtUrl := BuildURL("myPrefix:3000", "/myPath")
	assert.Equal(t, "http://myPrefix:3000/myPath", builtUrl)
}
