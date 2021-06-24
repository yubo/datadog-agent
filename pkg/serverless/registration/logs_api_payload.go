package registration

import (
	"encoding/json"
)

type Destination struct {
	URI      string `json:"URI"`
	Protocol string `json:"protocol"`
}
type Buffering struct {
	MaxBytes  int `json:"maxBytes"`
	MaxItems  int `json:"maxItems"`
	TimeoutMs int `json:"timeoutMs"`
}
type LogSubscriptionPayload struct {
	Buffering   Buffering   `json:"buffering"`
	Destination Destination `json:"destination"`
	Types       []string    `json:"types"`
}

func (p *LogSubscriptionPayload) MarshalJSON() ([]byte, error) {
	// use an alias to avoid infinite recursion while serializing
	type LogSubscriptionPayloadAlias LogSubscriptionPayload
	return json.Marshal((*LogSubscriptionPayloadAlias)(p))
}
