package snmp

import (
	"github.com/DataDog/datadog-agent/pkg/aggregator"
	"github.com/DataDog/datadog-agent/pkg/autodiscovery/integration"
	"github.com/DataDog/datadog-agent/pkg/collector/check"
	core "github.com/DataDog/datadog-agent/pkg/collector/corechecks"
	"github.com/DataDog/datadog-agent/pkg/util/log"
	"strings"
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
	log.Infof("columnValues: %#v\n", snmpValues.columnValues) // TODO: remove me

	// Submit metrics
	c.submitMetrics(snmpValues, tags)

	// Commit
	sender.Commit()
	return nil
}

func (c *Check) fetchValues(err error) (*snmpValues, error) {
	scalarResults, err := fetchScalarOids(c.session, c.config.OidConfig.scalarOids)
	if err != nil {
		log.Errorf("Get() err: %v", err)
		return &snmpValues{}, err
	}

	oids := make(map[string]string)
	for _, value := range c.config.OidConfig.columnOids {
		oids[value] = value
	}
	columnResults, err := fetchColumnOids(c.session, oids)
	if err != nil {
		log.Errorf("GetBulk() err: %v", err)
		return &snmpValues{}, err
	}

	return &snmpValues{scalarResults, columnResults}, nil
}

func (c *Check) submitMetrics(values *snmpValues, tags []string) {
	for _, metric := range c.config.Metrics {
		if metric.Symbol.OID != "" {
			c.submitScalarMetrics(metric, values, tags)
		} else if metric.Table.OID != "" {
			c.submitColumnMetrics(metric, values, tags)
		}
	}
}

func (c *Check) sendMetric(metric metricsConfig, metricName string, value float64, tags []string) {
	c.sender.Gauge("snmp."+metricName, value, "", tags)
}

func (c *Check) submitScalarMetrics(metric metricsConfig, values *snmpValues, tags []string) {
	value, err := values.getScalarFloat64(metric.Symbol.OID)
	if err != nil {
		log.Warnf("error getting scalar value: %v", err)
		return
	}
	c.sendMetric(metric, metric.Symbol.Name, value, tags)
}

func (c *Check) submitColumnMetrics(metricConfig metricsConfig, values *snmpValues, tags []string) {
	for _, symbol := range metricConfig.Symbols {
		values, err := values.getColumnValue(symbol.OID)
		if err != nil {
			log.Warnf("error getting column value: %v", err)
			continue
		}
		for fullIndex, value := range values {
			indexes := strings.Split(fullIndex, ".")
			rowTags := append(tags, metricConfig.getTags(indexes)...)
			c.sendMetric(metricConfig, symbol.Name, value, rowTags)
		}
		log.Infof("Table column %v - %v: %#v", symbol.Name, symbol.OID, values)
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
