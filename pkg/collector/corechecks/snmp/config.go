package snmp

import (
	"fmt"
	"github.com/DataDog/datadog-agent/pkg/autodiscovery/integration"
	"github.com/soniah/gosnmp"
	"gopkg.in/yaml.v2"
)

type snmpInitConfig struct {
	OidBatchSize             int `yaml:"oid_batch_size"`
	RefreshOidsCacheInterval int `yaml:"refresh_oids_cache_interval"`
	// TODO: To implement:
	// - global_metrics
	// - profiles
}

type snmpInstanceConfig struct {
	IPAddress       string `yaml:"ip_address"`
	Port            uint16 `yaml:"port"`
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

	// Related parse metric code: https://github.com/DataDog/integrations-core/blob/86e9dc09f5a1829a8e8bf1b404c4fb73a392e0e5/snmp/datadog_checks/snmp/parsing/metrics.py#L94-L150
	Metrics []metricsConfig `yaml:"metrics"`
	// TODO: To implement:
	//   - context_engine_id: Investigate if we can remove this configuration.
	//   - use_global_metrics
	//   - profile
	//   - metrics
	//   - metric_tags
}

type oidConfig struct {
	scalarOids []string
	columnOids []string
}

type snmpConfig struct {
	IPAddress       string
	Port            uint16
	CommunityString string
	SnmpVersion     gosnmp.SnmpVersion
	Timeout         int
	Retries         int
	User            string
	AuthProtocol    string
	AuthKey         string
	PrivProtocol    string
	PrivKey         string
	ContextName     string
	OidConfig       oidConfig
	Metrics         []metricsConfig
}

func buildConfig(rawInstance integration.Data, rawInitConfig integration.Data) (snmpConfig, error) {
	instance := snmpInstanceConfig{}
	init := snmpInitConfig{}

	err := yaml.Unmarshal(rawInitConfig, &init)
	if err != nil {
		return snmpConfig{}, err
	}

	err = yaml.Unmarshal(rawInstance, &instance)
	if err != nil {
		return snmpConfig{}, err
	}

	c := snmpConfig{}
	c.IPAddress = instance.IPAddress
	c.Port = instance.Port
	if instance.Port == 0 {
		c.Port = 161
	} else {
		c.Port = instance.Port
	}

	c.CommunityString = instance.CommunityString
	c.Metrics = instance.Metrics

	snmpVersion, err := parseVersion(instance.SnmpVersion)
	if err != nil {
		return snmpConfig{}, err
	}
	c.SnmpVersion = snmpVersion

	c.OidConfig.scalarOids = parseScalarOids(instance.Metrics)
	c.OidConfig.columnOids = parseColumnOids(instance.Metrics)

	return c, err
}

func parseScalarOids(metrics []metricsConfig) []string {
	var oids []string
	for _, metric := range metrics {
		if metric.Symbol.OID != "" { // TODO: test me
			oids = append(oids, metric.Symbol.OID)
		}
	}
	return oids
}

func parseColumnOids(metrics []metricsConfig) []string {
	var oids []string
	for _, metric := range metrics {
		if metric.Table.OID != "" { // TODO: test me
			for _, symbol := range metric.Symbols {
				oids = append(oids, symbol.OID)
			}
		}
	}
	return oids
}

func parseVersion(rawVersion string) (gosnmp.SnmpVersion, error) {
	switch rawVersion {
	case "1":
		return gosnmp.Version1, nil
	case "", "2", "2c":
		return gosnmp.Version2c, nil
	case "3":
		return gosnmp.Version3, nil
	}
	return 0, fmt.Errorf("invalid snmp version `%s`. Valid versions are: 1, 2, 2c, 3", rawVersion)
}
