package main

import (
	"strings"

	"github.com/DataDog/datadog-agent/cmd/agent/common"
	"github.com/DataDog/datadog-agent/pkg/autodiscovery"
	"github.com/DataDog/datadog-agent/pkg/logs"
	"github.com/DataDog/datadog-agent/pkg/serverless"
	"github.com/DataDog/datadog-agent/pkg/util/log"
)

const logsLogsTypeSubscribed = "DD_LOGS_CONFIG_LAMBDA_LOGS_TYPE"

func getLogTypesToSubscribe(envLogsType string) []string {
	var logsType []string
	if len(envLogsType) > 0 {
		var logsType []string
		parts := strings.Split(strings.TrimSpace(envLogsType), " ")
		for _, part := range parts {
			part = strings.ToLower(strings.TrimSpace(part))
			switch part {
			case "function", "platform", "extension":
				logsType = append(logsType, part)
			default:
				log.Warn("While subscribing to logs, unknown log type", part)
			}
		}
	} else {
		logsType = []string{"platform", "function", "extension"}
	}
	return logsType
}

func enableLogCollectionHttpRoute(daemon *serverless.Daemon) {
	log.Debug("Enabling logs collection HTTP route")
	if httpAddr, logsChan, err := daemon.EnableLogsCollection(); err != nil {
		log.Error("While starting the HTTP Logs Server:", err)
	} else {
		// subscribe to the logs on the platform
		if err := serverless.SubscribeLogs(serverlessID, httpAddr, logsType); err != nil {
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
