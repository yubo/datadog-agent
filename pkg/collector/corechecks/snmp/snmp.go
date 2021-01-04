package snmp

import (
	"fmt"
	"github.com/DataDog/datadog-agent/pkg/aggregator"
	"github.com/DataDog/datadog-agent/pkg/autodiscovery/integration"
	"github.com/DataDog/datadog-agent/pkg/collector/check"
	core "github.com/DataDog/datadog-agent/pkg/collector/corechecks"
	"github.com/DataDog/datadog-agent/pkg/metrics"
	"github.com/DataDog/datadog-agent/pkg/util/log"
	"path/filepath"
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

	tags := c.config.getGlobalTags()

	sender.MonotonicCount("snmp.check_interval", float64(time.Now().UnixNano())/1e9, "", tags)
	start := time.Now()

	c.sender = metricSender{sender: sender}

	// Create connection
	err = c.session.Connect()
	if err != nil {
		// TODO: Test connection error
		sender.ServiceCheck("snmp.can_check", metrics.ServiceCheckCritical, "", tags, err.Error())
		return fmt.Errorf("snmp connection error: %s", err)
	}
	// TODO: Handle service check tags like
	//   https://github.com/DataDog/integrations-core/blob/df2bc0d17af490491651d7578e67d9928941df62/snmp/datadog_checks/snmp/snmp.py#L401-L449
	sender.ServiceCheck("snmp.can_check", metrics.ServiceCheckOK, "", tags, "")
	defer c.session.Close() // TODO: handle error?

	if !c.config.OidConfig.hasOids() {
		sysObjectID, err := c.fetchSysObjectID()
		if err != nil {
			return fmt.Errorf("failed to fetching sysobjectid: %s", err)
		}
		profile, err := c.getProfileForSysObjectID(sysObjectID)
		if err != nil {
			return fmt.Errorf("failed to get profile sys object id for `%s`: %s", sysObjectID, err)
		}
		err = c.config.refreshWithProfile(profile)
		if err != nil {
			return fmt.Errorf("failed to refresh with profile: %s", err)
		}
		tags = c.config.getGlobalTags()
	}

	if c.config.OidConfig.hasOids() {
		c.config.addUptimeMetric()

		snmpValues, err := c.fetchValues(err)
		if err != nil {
			return err
		}

		// Report metrics
		tags = append(tags, c.sender.getGlobalMetricTags(c.config.MetricTags, snmpValues)...)
		c.sender.reportMetrics(c.config.Metrics, c.config.MetricTags, snmpValues, tags)
	}

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

func (c *Check) fetchSysObjectID() (string, error) {
	result, err := c.session.Get([]string{"1.3.6.1.2.1.1.2.0"})
	if err != nil {
		return "", fmt.Errorf("cannot get sysobjectid: %s", err)
	}
	return result.Variables[0].Value.(string), nil
}

func (c *Check) getProfileForSysObjectID(sysObjectID string) (string, error) {
	sysOidToProfile := map[string]string{}
	var matchedOids []string

	// TODO: Test me
	for profile, definition := range c.config.Profiles {
		// TODO: Check for duplicate profile sysobjectid
		//   https://github.com/DataDog/integrations-core/blob/df2bc0d17af490491651d7578e67d9928941df62/snmp/datadog_checks/snmp/snmp.py#L142-L144
		for _, oidPattern := range definition.SysObjectIds {

			found, err := filepath.Match(oidPattern, sysObjectID)
			if err != nil {
				log.Debugf("pattern error: %s", err)
				continue
			}
			if found {
				sysOidToProfile[oidPattern] = profile
				matchedOids = append(matchedOids, oidPattern)
			}
		}
	}
	oid, err := getMostSpecificOid(matchedOids)
	if err != nil {
		return "", fmt.Errorf("failed to get most specific oid, for matched oids %v: %s", matchedOids, err)
	}
	return sysOidToProfile[oid], nil
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
