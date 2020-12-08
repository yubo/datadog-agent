package snmp

import (
	"fmt"
	"github.com/DataDog/datadog-agent/pkg/aggregator"
	"github.com/DataDog/datadog-agent/pkg/autodiscovery/integration"
	"github.com/DataDog/datadog-agent/pkg/collector/check"
	core "github.com/DataDog/datadog-agent/pkg/collector/corechecks"
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

	c.sender = metricSender{sender}

	// Create connection
	err = c.session.Connect()
	if err != nil {
		// TODO: Test connection error
		return fmt.Errorf("snmp connection error: %v", err)
	}
	defer c.session.Close() // TODO: handle error?

	snmpValues, err := c.fetchValues(err)
	if err != nil {
		return err
	}

	// Report metrics
	c.sender.reportMetrics(c.config.Metrics, c.config.MetricTags, snmpValues, tags)

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
	err := c.CommonConfigure(rawInstance, source)
	if err != nil {
		return err
	}

	config, err := buildConfig(rawInstance, rawInitConfig)
	if err != nil {
		return err
	}

	c.BuildID(rawInstance, rawInitConfig)
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
