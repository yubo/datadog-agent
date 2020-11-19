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

	tags := []string{"snmp_device:" + c.config.IPAddress}

	c.sender = sender
	log.Infof("c.config.Metrics: %#v\n", c.config.Metrics) // TODO: remove me
	sender.Gauge("snmp.devices_monitored", float64(1), "", tags)

	// Create connection
	err = c.session.Connect()
	if err != nil {
		log.Errorf("Connect() err: %v", err)
	}
	defer c.session.Close() // TODO: handle error?

	snmpValues, err := c.fetchValues(err)
	if err != nil {
		return err
	}

	log.Infof("scalarValues: %#v\n", snmpValues.scalarValues) // TODO: remove me

	// Submit metrics
	c.submitMetrics(snmpValues, tags)

	// Commit
	sender.Commit()
	return nil
}

func (c *Check) fetchValues(err error) (*snmpValues, error) {
	log.Infof("Get() oids: %v", c.session, c.config.OidConfig.scalarOids)
	scalarResults, err := fetchScalarOids(c.session, c.config.OidConfig.scalarOids)
	if err != nil {
		log.Errorf("Get() err: %v", err)
		return &snmpValues{}, err
	}

	log.Infof("GetBulk() oids: %v", c.config.OidConfig.columnOids)

	columnResults, err := fetchColumnOids(c.session, c.config.OidConfig.columnOids)
	if err != nil {
		log.Errorf("GetBulk() err: %v", err)
		return &snmpValues{}, err
	}

	// Format scalarValues
	snmpValues := newSnmpValues()
	snmpValues.scalarValues = scalarResults
	snmpValues.columnValues = columnResults
	return snmpValues, nil
}

func (c *Check) submitMetrics(snmpValues *snmpValues, tags []string) {
	for _, metric := range c.config.Metrics {
		if metric.Symbol.OID != "" {
			value, ok := snmpValues.getScalarFloat64(metric.Symbol.OID)
			if ok {
				c.sender.Gauge("snmp."+metric.Symbol.Name, value, "", tags)
			}
		}
		if metric.Table.OID != "" {
			for _, symbol := range metric.Symbols {
				value := snmpValues.columnValues[symbol.OID]
				log.Infof("Table column %v - %v: %#v", symbol.Name, symbol.OID, value)
			}
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
