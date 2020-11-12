package snmp

import (
	"github.com/DataDog/datadog-agent/pkg/aggregator"
	"github.com/DataDog/datadog-agent/pkg/autodiscovery/integration"
	"github.com/DataDog/datadog-agent/pkg/collector/check"
	"github.com/DataDog/datadog-agent/pkg/util/log"
	"time"

	core "github.com/DataDog/datadog-agent/pkg/collector/corechecks"
	"github.com/soniah/gosnmp"
	yaml "gopkg.in/yaml.v2"
)

const (
	snmpCheckName = "snmp"
)

// Check aggregates metrics from one Check instance
type Check struct {
	core.CheckBase
	config snmpConfig
}

type snmpInitConfig struct {
	OidBatchSize             int `yaml:"oid_batch_size"`
	RefreshOidsCacheInterval int `yaml:"refresh_oids_cache_interval"`
	// TODO: To implement:
	// - global_metrics
	// - profiles
}

type snmpInstanceConfig struct {
	IPAddress       string `yaml:"ip_address"`
	Port            int    `yaml:"port"`
	CommunityString string `yaml:"community_string"`
	SnmpVersion     string `yaml:"snmp_version"`
	Timeout         int    `yaml:"timeout"`
	Retries         int    `yaml:"retries"`
	User            string `yaml:"user"`
	AuthProtocol    string `yaml:"authProtocol"`
	AuthKey         string `yaml:"authKey"`
	PrivProtocol    string `yaml:"privProtocol"`
	PrivKey         string `yaml:"privKey"`
	ContextName     string `yaml:"context_name"`
	// TODO: To implement:
	//   - context_engine_id: Investigate if we can remove this configuration.
	//   - use_global_metrics
	//   - profile
	//   - metrics
	//   - metric_tags
}

type snmpConfig struct {
	instance snmpInstanceConfig
	initConf snmpInitConfig
}

// Run executes the check
func (c *Check) Run() error {
	sender, err := aggregator.GetSender(c.ID())
	if err != nil {
		return err
	}

	sender.Gauge("snmp.test.metric", float64(10), "", nil)

	session := gosnmp.GoSNMP{
		Target:             "localhost",
		Port:               uint16(1161),
		Community:          "public",
		Version:            gosnmp.Version2c,
		Timeout:            time.Duration(2) * time.Second,
		Retries:            3,
		ExponentialTimeout: true,
		MaxOids:            100,
	}

	err = session.Connect()
	if err != nil {
		log.Errorf("Connect() err: %v", err)
	}
	defer session.Conn.Close()

	oids := []string{"1.3.6.1.2.1.25.6.3.1.5.130"}
	result, err := session.Get(oids)
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
		}
	}

	log.Debug("Run snmp")

	sender.Commit()

	return nil
}

// Configure configures the snmp checks
func (c *Check) Configure(rawInstance integration.Data, rawInitConfig integration.Data, source string) error {
	err := c.CommonConfigure(rawInstance, source)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(rawInitConfig, &c.config.initConf)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(rawInstance, &c.config.instance)
	if err != nil {
		return err
	}

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
