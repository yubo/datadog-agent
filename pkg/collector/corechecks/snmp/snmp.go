package snmp

import (
	"github.com/DataDog/datadog-agent/pkg/aggregator"
	"github.com/DataDog/datadog-agent/pkg/autodiscovery/integration"
	"github.com/DataDog/datadog-agent/pkg/collector/check"
	core "github.com/DataDog/datadog-agent/pkg/collector/corechecks"
	"github.com/DataDog/datadog-agent/pkg/util/log"
	"github.com/soniah/gosnmp"
)

const (
	snmpCheckName = "snmp"
)

// Check aggregates metrics from one Check instance
type Check struct {
	core.CheckBase
	config  snmpConfig
	session snmpSession
}

/*

- Parse configuration
	- Create Config object
- Parse metrics (profile + config)
	- Create Metrics list
		- List for
		- with tags
		- index transform
        - ...
	- Create OID->Symbol lookup map
	- Create list of OIDs to be fetched
		- Scalar OIDs
		- Table Column OIDs
- Create session
- Get OID values using session						https://github.com/DataDog/integrations-core/blob/74a3e98624b3cd39e76764ffeaeb279be7123cd5/snmp/datadog_checks/snmp/snmp.py#L191-L241
    - Separate OID and index parts
	- Store OID->value map
- Process metrics 									https://github.com/DataDog/integrations-core/blob/74a3e98624b3cd39e76764ffeaeb279be7123cd5/snmp/datadog_checks/snmp/snmp.py#L470-L501
	- Process each metric
	- Find correct index
    - And use fetched values
 */

// Run executes the check
func (c *Check) Run() error {
	sender, err := aggregator.GetSender(c.ID())
	if err != nil {
		return err
	}

	sender.Gauge("snmp.test.metric", float64(10), "", nil)

	err = c.session.Connect()
	if err != nil {
		log.Errorf("Connect() err: %v", err)
	}
	defer c.session.Close() // TODO: handle error?

	for _, metric := range c.config.Metrics {

		oids := []string{metric.OID}
		result, err := c.session.Get(oids)
		if err != nil {
			log.Errorf("Get() err: %v", err)
			return nil
		}

		for j, variable := range result.Variables {
			log.Infof("%d: oid: %s ", j, variable.Name)
			switch variable.Type {
			case gosnmp.OctetString:
				log.Infof("string: %s\n", string(variable.Value.([]byte)))
			default:
				log.Infof("number: %d\n", gosnmp.ToBigInt(variable.Value))
				value := float64(gosnmp.ToBigInt(variable.Value).Int64())
				sender.Gauge("snmp."+metric.Name, value, "", nil)
			}
		}
	}

	sender.Commit()

	return nil
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
	c.session = buildSession(c.config)

	return nil
}

func snmpFactory() check.Check {
	return &Check{
		CheckBase: core.NewCheckBase(snmpCheckName),
	}
}

func init() {
	core.RegisterCheck(snmpCheckName, snmpFactory)
}
