package snmp

import (
	"fmt"
	"github.com/DataDog/datadog-agent/pkg/aggregator"
	"github.com/DataDog/datadog-agent/pkg/autodiscovery/integration"
	"github.com/DataDog/datadog-agent/pkg/collector/check"
	core "github.com/DataDog/datadog-agent/pkg/collector/corechecks"
	"github.com/DataDog/datadog-agent/pkg/metrics"
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
	sender, err := aggregator.GetSender(c.ID())
	if err != nil {
		return err
	}

	tags := []string{"snmp_device:" + c.config.IPAddress}
	tags = append(tags, c.config.Tags...)

	// TODO: Remove Telemetry
	tags = append(tags, "loader:core")
	sender.Rate("snmp.check_interval", float64(time.Now().UnixNano())/1e9, "", tags)
	start := time.Now()

	c.sender = metricSender{sender: sender}

	// Create connection
	err = c.session.Connect()
	if err != nil {
		// TODO: Test connection error
		sender.ServiceCheck("snmp.can_check", metrics.ServiceCheckCritical, "", tags, err.Error())
		return fmt.Errorf("snmp connection error: %v", err)
	}
	sender.ServiceCheck("snmp.can_check", metrics.ServiceCheckOK, "", tags, "")
	defer c.session.Close() // TODO: handle error?

	snmpValues, err := c.fetchValues(err)
	if err != nil {
		return err
	}

	// Report metrics
	c.sender.reportMetrics(c.config.Metrics, c.config.MetricTags, snmpValues, tags)

	// TODO: Remove Telemetry
	sender.Gauge("snmp.check_duration", float64(time.Since(start))/1e9, "", tags)
	sender.Gauge("snmp.submitted_metrics", float64(c.sender.submittedMetrics), "", tags)

	// Commit
	sender.Commit()
	return nil
}

func (c *Check) fetchValues(err error) (*snmpValues, error) {
	scalarResults, err := fetchScalarOidsByBatch(c.session, c.config.OidConfig.scalarOids, c.config.OidBatchSize)
	if err != nil {
		return &snmpValues{}, fmt.Errorf("SNMPGET error: %v", err)
	}

	oids := make(map[string]string)
	for _, value := range c.config.OidConfig.columnOids {
		oids[value] = value
	}
	columnResults, err := fetchColumnOids(c.session, oids, c.config.OidBatchSize)
	if err != nil {
		return &snmpValues{}, fmt.Errorf("SNMPBULK error: %v", err)
	}

	return &snmpValues{scalarResults, columnResults}, nil
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

	c.config = config
	c.session.Configure(c.config)

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
