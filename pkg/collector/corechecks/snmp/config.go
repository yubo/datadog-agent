package snmp

import (
	"fmt"
	"github.com/DataDog/datadog-agent/pkg/autodiscovery/integration"
	"github.com/soniah/gosnmp"
	"gopkg.in/yaml.v2"
)

var defaultOidBatchSize = 10

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
	Metrics    []metricsConfig   `yaml:"metrics"`
	MetricTags []metricTagConfig `yaml:"metric_tags"`

	Profiles profilesConfig `yaml:"profiles"`
	Profile  string         `yaml:"profile"`

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
	MetricTags      []metricTagConfig
	OidBatchSize    int
	Profiles        profileDefinitionMap
	Tags            []string
}

func (c *snmpConfig) refreshWithProfile(definition profileDefinition) {
	// https://github.com/DataDog/integrations-core/blob/e64e2d18529c6c106f02435c5fdf2621667c16ad/snmp/datadog_checks/snmp/config.py#L181-L200
	c.Metrics = append(c.Metrics, definition.Metrics...)
	c.MetricTags = append(c.MetricTags, definition.MetricTags...)
	c.OidConfig.scalarOids = append(c.OidConfig.scalarOids, parseScalarOids(definition.Metrics, definition.MetricTags)...)
	c.OidConfig.columnOids = append(c.OidConfig.columnOids, parseColumnOids(definition.Metrics)...)

	if definition.Device.Vendor != "" {
		c.Tags = append(c.Tags, "device_vendor:"+definition.Device.Vendor)
	}

	//device = profile['definition'].get('device', {})
	//self.add_device_tags(device)
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

	c.Metrics = append(c.Metrics, getUptimeMetricConfig())
	c.MetricTags = instance.MetricTags

	snmpVersion, err := parseVersion(instance.SnmpVersion)
	if err != nil {
		return snmpConfig{}, err
	}
	c.SnmpVersion = snmpVersion

	// TODO: test me
	c.OidConfig.scalarOids = parseScalarOids(c.Metrics, c.MetricTags)
	c.OidConfig.columnOids = parseColumnOids(c.Metrics)

	if init.OidBatchSize == 0 {
		c.OidBatchSize = defaultOidBatchSize
	} else {
		c.OidBatchSize = init.OidBatchSize
	}

	profiles, err := loadProfiles(instance.Profiles)
	if err != nil {
		return snmpConfig{}, err
	}
	c.Profiles = profiles
	profile := instance.Profile

	if profile != "" {
		if _, ok := c.Profiles[profile]; !ok {
			return snmpConfig{}, fmt.Errorf("unknown profile '%s'", profile)
		}
		c.Tags = append(c.Tags, "profile:"+profile)
		c.refreshWithProfile(c.Profiles[profile])
	}

	return c, err
}

func getUptimeMetricConfig() metricsConfig {
	// Reference sysUpTimeInstance directly, see http://oidref.com/1.3.6.1.2.1.1.3.0
	return metricsConfig{Symbol: symbolConfig{OID: "1.3.6.1.2.1.1.3.0", Name: "sysUpTimeInstance"}}
}

func parseScalarOids(metrics []metricsConfig, metricTags []metricTagConfig) []string {
	var oids []string
	for _, metric := range metrics {
		if metric.Symbol.OID != "" { // TODO: test me
			oids = append(oids, metric.Symbol.OID)
		}
	}
	for _, metricTag := range metricTags {
		if metricTag.OID != "" { // TODO: test me
			oids = append(oids, metricTag.OID)
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
			for _, metricTag := range metric.MetricTags {
				if metricTag.Column.OID != "" {
					oids = append(oids, metricTag.Column.OID)
				}
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
