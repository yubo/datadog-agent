package main

import (
	"os"
	"time"

	"github.com/DataDog/datadog-agent/cmd/agent/common"
	"github.com/DataDog/datadog-agent/pkg/autodiscovery"
	"github.com/DataDog/datadog-agent/pkg/logs"
	"github.com/DataDog/datadog-agent/pkg/serverless"
	"github.com/DataDog/datadog-agent/pkg/serverless/registration"
	"github.com/DataDog/datadog-agent/pkg/util/log"
)

const logsLogsTypeSubscribed = "DD_LOGS_CONFIG_LAMBDA_LOGS_TYPE"

func enableLogCollectionHttpRoute(daemon *serverless.Daemon, serverlessID registration.ID, url string, logsType []string) {
	log.Debug("Enabling logs collection HTTP route")
	if httpAddr, logsChan, err := daemon.EnableLogsCollection(); err != nil {
		log.Error("While starting the HTTP Logs Server:", err)
	} else {
		// subscribe to the logs on the platform
		payload := registration.BuildLogRegistrationPayload(url, os.Getenv(logsLogsTypeSubscribed), 1000, 262144, 1000)
		if err := registration.SubscribeLogs(serverlessID, url, 5*time.Second, payload); err != nil {
			log.Error("Can't subscribe to logs:", err)
		} else {
			// we subscribed to the logs collection on the platform, let's instantiate
			// a logs agent to collect/process/flush the logs.
			if err := logs.StartServerless(
				func() *autodiscovery.AutoConfig { return common.AC },
				logsChan, nil,
			); err != nil {
				log.Error("Could not start an instance of the Logs Agent:", err)
			}
		}
	}
}
