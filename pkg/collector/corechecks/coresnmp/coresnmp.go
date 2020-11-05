package coresnmp

import (
	"github.com/DataDog/datadog-agent/pkg/aggregator"
	"github.com/DataDog/datadog-agent/pkg/autodiscovery/integration"
	"github.com/DataDog/datadog-agent/pkg/collector/check"
	"github.com/DataDog/datadog-agent/pkg/util/log"
	"time"

	core "github.com/DataDog/datadog-agent/pkg/collector/corechecks"
	yaml "gopkg.in/yaml.v2"
	"github.com/soniah/gosnmp"
)

const (
	coresnmpCheckName = "coresnmp"

)

// metricConfigItem map a metric to a coresnmp unit property.
type metricConfigItem struct {
	metricName         string
	propertyName       string
	accountingProperty string
	optional           bool // if optional log as debug when there is an issue getting the property, otherwise log as error
}

// CoresnmpCheck aggregates metrics from one CoresnmpCheck instance
type CoresnmpCheck struct {
	core.CheckBase
	config coresnmpConfig
}
type unitSubstateMapping = map[string]string

type systemdInstanceConfig struct {
	PrivateSocket         string                         `yaml:"private_socket"`
	UnitNames             []string                       `yaml:"unit_names"`
	SubstateStatusMapping map[string]unitSubstateMapping `yaml:"substate_status_mapping"`
}

type systemdInitConfig struct{}

type coresnmpConfig struct {
	instance systemdInstanceConfig
	initConf systemdInitConfig
}


// Run executes the check
func (c *CoresnmpCheck) Run() error {
	sender, err := aggregator.GetSender(c.ID())
	if err != nil {
		return err
	}

	sender.Gauge("coresnmp.test.metric", float64(10), "", nil)

	session := gosnmp.GoSNMP{
		Target: "localhost",
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

	log.Debug("Run coresnmp")

	sender.Commit()

	return nil
}

// Configure configures the coresnmp checks
func (c *CoresnmpCheck) Configure(rawInstance integration.Data, rawInitConfig integration.Data, source string) error {
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

func coresnmpFactory() check.Check {
	return &CoresnmpCheck{
		CheckBase: core.NewCheckBase(coresnmpCheckName),
	}
}

func init() {
	core.RegisterCheck(coresnmpCheckName, coresnmpFactory)
}
