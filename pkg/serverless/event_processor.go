package serverless

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/DataDog/datadog-agent/pkg/metrics"
	"github.com/DataDog/datadog-agent/pkg/serverless/registration"
	"github.com/DataDog/datadog-agent/pkg/util/log"
)

const (
	routeEventNext string = "/2020-01-01/extension/event/next"
)

// Payload is the payload read in the response while subscribing to
// the AWS Extension env.
type Payload struct {
	EventType          RuntimeEvent   `json:"eventType"`
	DeadlineMs         int64          `json:"deadlineMs"`
	InvokedFunctionArn string         `json:"invokedFunctionArn"`
	ShutdownReason     ShutdownReason `json:"shutdownReason"`
	//    RequestId string `json:"requestId"` // unused
}

// WaitForNextEvent makes a blocking HTTP call to receive the next event from AWS.
// Note that for now, we only subscribe to INVOKE and SHUTDOWN events.
// Write into stopCh to stop the main thread of the running program.
func WaitForNextEvent(stopCh chan struct{}, daemon *Daemon, metricsChan chan []metrics.MetricSample, id registration.ID, coldstart bool, prefix string) error {
	var err error
	var request *http.Request
	var response *http.Response

	if request, err = http.NewRequest(http.MethodGet, registration.BuildURL(prefix, routeEventNext), nil); err != nil {
		return fmt.Errorf("WaitForNextInvocation: can't create the GET request: %v", err)
	}
	request.Header.Set(registration.HeaderExtID, id.String())

	// make a blocking HTTP call to wait for the next event from AWS
	log.Debug("Waiting for next invocation...")
	client := &http.Client{Timeout: 0} // this one should never timeout
	if response, err = client.Do(request); err != nil {
		return fmt.Errorf("WaitForNextInvocation: while GET next route: %v", err)
	}
	daemon.StartInvocation()

	// we received an INVOKE or SHUTDOWN event
	daemon.StoreInvocationTime(time.Now())

	var body []byte
	if body, err = ioutil.ReadAll(response.Body); err != nil {
		return fmt.Errorf("WaitForNextInvocation: can't read the body: %v", err)
	}
	defer response.Body.Close()

	var payload Payload
	if err := json.Unmarshal(body, &payload); err != nil {
		return fmt.Errorf("WaitForNextInvocation: can't unmarshal the payload: %v", err)
	}

	if payload.EventType == Invoke {
		callInvocationHandler(daemon, payload.InvokedFunctionArn, payload.DeadlineMs, safetyBufferTimeout, coldstart, handleInvocation)
	}
	if payload.EventType == Shutdown {
		log.Debug("Received shutdown event. Reason: " + payload.ShutdownReason)
		isTimeout := strings.ToLower(payload.ShutdownReason.String()) == Timeout.String()
		if isTimeout {
			metricTags := addColdStartTag(daemon.extraTags)
			sendTimeoutEnhancedMetric(metricTags, metricsChan)
		}
		daemon.Stop(isTimeout)
		stopCh <- struct{}{}
	}
	return nil
}
