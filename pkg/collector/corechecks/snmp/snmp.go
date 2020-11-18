package snmp

import (
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
	config  snmpConfig
	session sessionAPI
	sender  aggregator.Sender
}

// Run executes the check
func (c *Check) Run() error {
	sender, err := aggregator.GetSender(c.ID())
	if err != nil {
		return err
	}

	c.sender = sender
	log.Infof("c.config.Metrics: %#v\n", c.config.Metrics) // TODO: remove me
	sender.Gauge("snmp.devices_monitored", float64(1), "", nil)

	// Create connection
	err = c.session.Connect()
	if err != nil {
		log.Errorf("Connect() err: %v", err)
	}
	defer c.session.Close() // TODO: handle error?

	// Get results
	result, err := c.session.Get(c.config.OidConfig.scalarOids)
	if err != nil {
		log.Errorf("Get() err: %v", err)
		return nil
	}

	// Format values
	snmpValues := resultToValues(result)
	log.Infof("values: %#v\n", snmpValues.values) // TODO: remove me

	// Submit metrics
	c.submitMetrics(snmpValues)

	// Commit
	sender.Commit()
	return nil
}

func (c *Check) submitMetrics(snmpValues snmpValues) {
	for _, metric := range c.config.Metrics {
		value, ok := snmpValues.getFloat64(metric.OID)
		if ok {
			c.sender.Gauge("snmp."+metric.Name, value, "", nil)
		}
	}
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
