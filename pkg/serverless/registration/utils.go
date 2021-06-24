package registration

import (
	"fmt"
	"net/http"
)

// ID is the extension ID within the AWS Extension environment.
type ID string

// String returns the string value for this ID.
func (i ID) String() string {
	return string(i)
}

// HttpClient represents an Http Client
type HttpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func BuildURL(prefix string, route string) string {
	if len(prefix) == 0 {
		return fmt.Sprintf("http://localhost:9001%s", route)
	}
	return fmt.Sprintf("http://%s%s", prefix, route)
}
