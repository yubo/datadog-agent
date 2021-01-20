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

	staticTags := c.config.getStaticTags()

	var checkErr error
	tags, checkErr := c.doRun(staticTags)
	if checkErr != nil {
		c.sender.serviceCheck("snmp.can_check", metrics.ServiceCheckCritical, "", tags, checkErr.Error())
	} else {
		c.sender.serviceCheck("snmp.can_check", metrics.ServiceCheckOK, "", tags, "")
	}

	c.sender.gauge("snmp.devices_monitored", float64(1), "", tags)

	// SNMP Performance metrics
	c.sender.monotonicCount("datadog.snmp.check_interval", float64(time.Now().UnixNano())/1e9, "", tags)
	c.sender.gauge("datadog.snmp.check_duration", float64(time.Since(start))/1e9, "", tags)
	c.sender.gauge("datadog.snmp.submitted_metrics", float64(c.sender.submittedMetrics), "", tags)

	// Commit
	sender.Commit()
	return checkErr
}

func (c *Check) doRun(staticTags []string) (retTags []string, retErr error) {
	retTags = copyTags(staticTags)

	// Create connection
	connErr := c.session.Connect()
	if connErr != nil {
		retErr = fmt.Errorf("snmp connection error: %s", connErr)
		return
	}
	defer func() {
		err := c.session.Close()
		if err != nil && retErr != nil {
			retErr = err
		}
	}()

	// If no OIDs, try to detect profile using device sysobjectid
	if !c.config.oidConfig.hasOids() {
		sysObjectID, err := fetchSysObjectID(c.session)
		if err != nil {
			retErr = fmt.Errorf("failed to fetching sysobjectid: %s", err)
			return
		}
		profile, err := getProfileForSysObjectID(c.config.profiles, sysObjectID)
		if err != nil {
			retErr = fmt.Errorf("failed to get profile sys object id for `%s`: %s", sysObjectID, err)
			return
		}
		err = c.config.refreshWithProfile(profile)
		if err != nil {
			// Should not happen since the profile is one of those we matched in getProfileForSysObjectID
			retErr = fmt.Errorf("failed to refresh with profile `%s` detected using sysObjectID `%s`: %s", profile, sysObjectID, err)
			return
		}
	}
	retTags = append(retTags, c.config.profileTags...)

	// Fetch and report metrics
	if c.config.oidConfig.hasOids() {
		c.config.addUptimeMetric()

		valuesStore, err := fetchValues(c.session, c.config)
		if err != nil {
			retErr = fmt.Errorf("failed to fetch values: %s", err)
			return
		}
		log.Debugf("fetched valuesStore: %#v", valuesStore)
		retTags = append(retTags, c.sender.getCheckInstanceMetricTags(c.config.metricTags, valuesStore)...)
		c.sender.reportMetrics(c.config.metrics, valuesStore, retTags)
	}
	return
}

// Configure configures the snmp checks
func (c *Check) Configure(rawInstance integration.Data, rawInitConfig integration.Data, source string) error {
	// Must be called before c.CommonConfigure
	c.BuildID(rawInstance, rawInitConfig)

	err := c.CommonConfigure(rawInstance, source)
	if err != nil {
		return fmt.Errorf("common configure failed: %s", err)
	}

	config, err := buildConfig(rawInstance, rawInitConfig)
	if err != nil {
		return fmt.Errorf("build config failed: %s", err)
	}

	log.Debugf("SNMP configuration: %s", config.toString())

	c.config = config
	err = c.session.Configure(c.config)
	if err != nil {
		return fmt.Errorf("session configure failed: %s", err)
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
