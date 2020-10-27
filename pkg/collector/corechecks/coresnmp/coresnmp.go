package coresnmp

import (

	"github.com/DataDog/datadog-agent/pkg/aggregator"
	"github.com/DataDog/datadog-agent/pkg/autodiscovery/integration"
	"github.com/DataDog/datadog-agent/pkg/collector/check"
	"github.com/DataDog/datadog-agent/pkg/util/log"

	core "github.com/DataDog/datadog-agent/pkg/collector/corechecks"
	yaml "gopkg.in/yaml.v2"
)

const (
	coresnmpCheckName = "coresnmp"

)

// metricConfigItem map a metric to a systemd unit property.
type metricConfigItem struct {
	metricName         string
	propertyName       string
	accountingProperty string
	optional           bool // if optional log as debug when there is an issue getting the property, otherwise log as error
}

// CoresnmpCheck aggregates metrics from one CoresnmpCheck instance
type CoresnmpCheck struct {
	core.CheckBase
	config systemdConfig
}
type unitSubstateMapping = map[string]string

type systemdInstanceConfig struct {
	PrivateSocket         string                         `yaml:"private_socket"`
	UnitNames             []string                       `yaml:"unit_names"`
	SubstateStatusMapping map[string]unitSubstateMapping `yaml:"substate_status_mapping"`
}

type systemdInitConfig struct{}

type systemdConfig struct {
	instance systemdInstanceConfig
	initConf systemdInitConfig
}


// Run executes the check
func (c *CoresnmpCheck) Run() error {
	sender, err := aggregator.GetSender(c.ID())
	if err != nil {
		return err
	}

	log.Debug("Run coresnmp")

	sender.Commit()

	return nil
}

// Configure configures the systemd checks
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
