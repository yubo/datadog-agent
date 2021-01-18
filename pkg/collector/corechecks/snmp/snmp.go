package snmp

import (
	"fmt"
	"github.com/DataDog/datadog-agent/pkg/aggregator"
	"github.com/DataDog/datadog-agent/pkg/autodiscovery/integration"
	"github.com/DataDog/datadog-agent/pkg/collector/check"
	core "github.com/DataDog/datadog-agent/pkg/collector/corechecks"
	"github.com/DataDog/datadog-agent/pkg/metrics"
	"github.com/DataDog/datadog-agent/pkg/util/log"
	"time"
)

const (
	snmpCheckName = "snmp"
)

// Check aggregates metrics from one Check instance
type Check struct {
	core.CheckBase
	config  snmpConfig
	session sessionAPI
	sender  metricSender
}

// Run executes the check
func (c *Check) Run() error {
	start := time.Now()

	sender, err := aggregator.GetSender(c.ID())
	if err != nil {
		return err
	}
	c.sender = metricSender{sender: sender}

	tags := c.config.getInstanceTags()

	// Create connection
	err = c.session.Connect()
	if err != nil {
		// TODO: Test connection error
		sender.ServiceCheck("snmp.can_check", metrics.ServiceCheckCritical, "", copyTags(tags), err.Error())
		return fmt.Errorf("snmp connection error: %s", err)
	}
	// TODO: Handle service check tags like
	//   https://github.com/DataDog/integrations-core/blob/df2bc0d17af490491651d7578e67d9928941df62/snmp/datadog_checks/snmp/snmp.py#L401-L449
	sender.ServiceCheck("snmp.can_check", metrics.ServiceCheckOK, "", copyTags(tags), "")
	defer c.session.Close() // TODO: handle error?

	// If no OIDs, try to detect profile using device sysobjectid
	if !c.config.OidConfig.hasOids() {
		sysObjectID, err := fetchSysObjectID(c.session)
		if err != nil {
			return fmt.Errorf("failed to fetching sysobjectid: %s", err)
		}
		profile, err := getProfileForSysObjectID(c.config.Profiles, sysObjectID)
		if err != nil {
			return fmt.Errorf("failed to get profile sys object id for `%s`: %s", sysObjectID, err)
		}
		err = c.config.refreshWithProfile(profile)
		if err != nil {
			return fmt.Errorf("failed to refresh with profile: %s", err)
		}
		tags = c.config.getInstanceTags()
	}

	// Fetch and report metrics
	if c.config.OidConfig.hasOids() {
		c.config.addUptimeMetric()

		snmpValues, err := fetchValues(c.session, c.config)
		if err != nil {
			return err
		}
		log.Debugf("fetched snmpValues: %#v", snmpValues)
		tags = append(tags, c.sender.getCheckInstanceMetricTags(c.config.MetricTags, snmpValues)...)
		c.sender.reportMetrics(c.config.Metrics, snmpValues, tags)
	}

	// SNMP Performance metrics
	// TODO: Remove Telemetry?
	sender.MonotonicCount("datadog.snmp.check_interval", float64(time.Now().UnixNano())/1e9, "", copyTags(tags))
	sender.Gauge("datadog.snmp.check_duration", float64(time.Since(start))/1e9, "", copyTags(tags))
	sender.Gauge("datadog.snmp.submitted_metrics", float64(c.sender.submittedMetrics), "", copyTags(tags))

	// Commit
	sender.Commit()
	return nil
}

// Configure configures the snmp checks
func (c *Check) Configure(rawInstance integration.Data, rawInitConfig integration.Data, source string) error {
	// Must be called before c.CommonConfigure
	c.BuildID(rawInstance, rawInitConfig)

	// TODO: test that instance tags are passed correctly to sender
	err := c.CommonConfigure(rawInstance, source)
	if err != nil {
		return err
	}

	config, err := buildConfig(rawInstance, rawInitConfig)
	if err != nil {
		return err
	}

	// TODO: Clean up sensitive info: community string, auth key, priv key
	log.Debugf("Config: %#v", config)

	c.config = config
	err = c.session.Configure(c.config)
	if err != nil {
		return err
	}

	return nil
}

func snmpFactory() check.Check {
	return &Check{
		session:   &snmpSession{},
		CheckBase: core.NewCheckBase(snmpCheckName),
	}
}

func init() {
	core.RegisterCheck(snmpCheckName, snmpFactory)
}
