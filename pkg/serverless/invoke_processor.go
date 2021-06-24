package serverless

import (
	"context"
	"time"

	"github.com/DataDog/datadog-agent/pkg/config"
	"github.com/DataDog/datadog-agent/pkg/serverless/aws"
	"github.com/DataDog/datadog-agent/pkg/serverless/flush"
	"github.com/DataDog/datadog-agent/pkg/util/log"
)

const (
	clientReadyTimeout  time.Duration = 2 * time.Second
	safetyBufferTimeout time.Duration = 20 * time.Millisecond
)

// InvocationHandler is the invocation handler signature
type InvocationHandler func(doneChannel chan bool, daemon *Daemon, arn string, coldstart bool)

func callInvocationHandler(daemon *Daemon, arn string, deadlineMs int64, safetyBufferTimeout time.Duration, coldstart bool, invocationHandler InvocationHandler) {
	timeout := computeTimeout(time.Now(), deadlineMs, safetyBufferTimeout)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	doneChannel := make(chan bool)
	go invocationHandler(doneChannel, daemon, arn, coldstart)
	select {
	case <-ctx.Done():
		log.Debug("Timeout detected, finishing the current invocation now to allow receiving the SHUTDOWN event")
		daemon.FinishInvocation()
		return
	case <-doneChannel:
		return
	}
}

func handleInvocation(doneChannel chan bool, daemon *Daemon, arn string, coldstart bool) {
	log.Debug("Received invocation event...")
	daemon.ComputeGlobalTags(arn, config.GetConfiguredTags(true))
	aws.SetARN(arn)
	if coldstart {
		ready := daemon.WaitUntilClientReady(clientReadyTimeout)
		if ready {
			log.Debug("Client library registered with extension")
		} else {
			log.Debug("Timed out waiting for client library to register with extension.")
		}
		daemon.UpdateStrategy()
	}

	// immediately check if we should flush data
	// note that since we're flushing synchronously here, there is a scenario
	// where this could be blocking the function if the flush is slow (if the
	// extension is not quickly going back to listen on the "wait next event"
	// route). That's why we use a context.Context with a timeout `flushTimeout``
	// to avoid blocking for too long.
	// This flushTimeout is re-using the forwarder_timeout value.
	if daemon.flushStrategy.ShouldFlush(flush.Starting, time.Now()) {
		log.Debugf("The flush strategy %s has decided to flush the data in the moment: %s", daemon.flushStrategy, flush.Starting)
		flushTimeout := config.Datadog.GetDuration("forwarder_timeout") * time.Second
		ctx, cancel := context.WithTimeout(context.Background(), flushTimeout)
		daemon.TriggerFlush(ctx, false)
		cancel() // free the resource of the context
	} else {
		log.Debugf("The flush strategy %s has decided to not flush in the moment: %s", daemon.flushStrategy, flush.Starting)
	}
	daemon.WaitForDaemon()
	doneChannel <- true
}

func computeTimeout(now time.Time, deadlineMs int64, safetyBuffer time.Duration) time.Duration {
	currentTimeInMs := now.UnixNano() / int64(time.Millisecond)
	return time.Duration((deadlineMs-currentTimeInMs)*int64(time.Millisecond) - int64(safetyBuffer))
}
