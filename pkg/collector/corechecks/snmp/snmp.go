package snmp

import (
	"fmt"
	"github.com/DataDog/datadog-agent/pkg/metrics"
	"os"
	"strconv"
	"time"

	"github.com/DataDog/datadog-agent/pkg/aggregator"
	"github.com/DataDog/datadog-agent/pkg/autodiscovery/integration"
	"github.com/DataDog/datadog-agent/pkg/collector/check"
	core "github.com/DataDog/datadog-agent/pkg/collector/corechecks"
	"github.com/DataDog/datadog-agent/pkg/util/log"
)

const (
	snmpCheckName = "snmp"
)

// Check aggregates metrics from one Check instance
type Check struct {
	core.CheckBase
	config          snmpConfig
	session         sessionAPI
	sender          metricSender
	sendFakeMetrics bool
	sleepTime       time.Duration
	metricsToSend   int
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
	tags := copyStrings(staticTags)

	var retErr error
	if c.sendFakeMetrics {
		time.Sleep(c.sleepTime)
		for i := 0; i < c.metricsToSend; i++ {
			c.sender.sendMetric("snmp.my_metric"+strconv.Itoa(i), snmpValueType{"gauge", "1"}, staticTags, "counter", metricsConfigOption{})
		}
	} else {
		metricTags, checkErr := c.processSnmpMetrics(staticTags)
		tags = metricTags
		retErr = checkErr
		if checkErr != nil {
			c.sender.serviceCheck("snmp.can_check", metrics.ServiceCheckCritical, "", tags, checkErr.Error())
		} else {
			c.sender.serviceCheck("snmp.can_check", metrics.ServiceCheckOK, "", tags, "")
		}
	}

	c.sender.gauge("snmp.devices_monitored", float64(1), "", tags)

	// SNMP Performance metrics
	c.sender.monotonicCount("datadog.snmp.check_interval", time.Duration(start.UnixNano()).Seconds(), "", tags)
	c.sender.gauge("datadog.snmp.check_duration", time.Since(start).Seconds(), "", tags)
	c.sender.gauge("datadog.snmp.submitted_metrics", float64(c.sender.submittedMetrics), "", tags)

	// Commit
	sender.Commit()
	return retErr
}

func (c *Check) processSnmpMetrics(staticTags []string) ([]string, error) {
	tags := copyStrings(staticTags)

	// Create connection
	connErr := c.session.Connect()
	if connErr != nil {
		return tags, fmt.Errorf("snmp connection error: %s", connErr)
	}
	defer func() {
		err := c.session.Close()
		if err != nil {
			log.Warnf("failed to close session: %v", err)
		}
	}()

	// If no OIDs, try to detect profile using device sysobjectid
	if !c.config.oidConfig.hasOids() {
		sysObjectID, err := fetchSysObjectID(c.session)
		if err != nil {
			return tags, fmt.Errorf("failed to fetching sysobjectid: %s", err)
		}
		profile, err := getProfileForSysObjectID(c.config.profiles, sysObjectID)
		if err != nil {
			return tags, fmt.Errorf("failed to get profile sys object id for `%s`: %s", sysObjectID, err)
		}
		err = c.config.refreshWithProfile(profile)
		if err != nil {
			// Should not happen since the profile is one of those we matched in getProfileForSysObjectID
			return tags, fmt.Errorf("failed to refresh with profile `%s` detected using sysObjectID `%s`: %s", profile, sysObjectID, err)
		}
	}
	tags = append(tags, c.config.profileTags...)

	// Fetch and report metrics
	if c.config.oidConfig.hasOids() {
		c.config.addUptimeMetric()

		valuesStore, err := fetchValues(c.session, c.config)
		if err != nil {
			return tags, fmt.Errorf("failed to fetch values: %s", err)
		}
		log.Debugf("fetched valuesStore: %#v", valuesStore)
		tags = append(tags, c.sender.getCheckInstanceMetricTags(c.config.metricTags, valuesStore)...)
		c.sender.reportMetrics(c.config.metrics, valuesStore, tags)
	}
	return tags, nil
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

	c.sendFakeMetrics = os.Getenv("DD_SEND_FAKE_METRICS") == "true"

	sleepTimeStr := os.Getenv("DD_SLEEP_TIME")
	if sleepTimeStr != "" {
		sleepTime, err := strconv.Atoi(sleepTimeStr)
		if err != nil {
			return fmt.Errorf("error getting DD_SLEEP_TIME: %s", err)
		}
		c.sleepTime = time.Duration(sleepTime) * time.Second
	}

	metricsToSendStr := os.Getenv("DD_METRICS_TO_SEND")
	if metricsToSendStr != "" {
		metricsToSend, err := strconv.Atoi(metricsToSendStr)
		if err != nil {
			return fmt.Errorf("error getting DD_METRICS_TO_SEND: %s", err)
		}
		c.metricsToSend = metricsToSend
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
