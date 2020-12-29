package snmp

import (
	"github.com/DataDog/datadog-agent/pkg/config"
	assert "github.com/stretchr/testify/require"
	"path/filepath"
	"testing"
)

func mockProfilesDefinitions() profileDefinitionMap {
	metrics := []metricsConfig{
		{Symbol: symbolConfig{OID: "1.3.6.1.4.1.3375.2.1.1.2.1.44.0", Name: "sysStatMemoryTotal"}, ForcedType: "gauge"},
		{
			Table:      symbolConfig{OID: "1.3.6.1.2.1.2.2", Name: "ifTable"},
			ForcedType: "monotonic_count",
			Symbols: []symbolConfig{
				{OID: "1.3.6.1.2.1.2.2.1.14", Name: "ifInErrors"},
				{OID: "1.3.6.1.2.1.2.2.1.13", Name: "ifInDiscards"},
			},
			MetricTags: []metricTagConfig{
				{Tag: "interface", Column: symbolConfig{OID: "1.3.6.1.2.1.31.1.1.1.1", Name: "ifName"}},
				{Tag: "interface_alias", Column: symbolConfig{OID: "1.3.6.1.2.1.31.1.1.1.18", Name: "ifAlias"}},
			},
		},
		{Symbol: symbolConfig{OID: "1.2.3.4.5", Name: "someMetric"}},
	}
	return profileDefinitionMap{"f5-big-ip": profileDefinition{
		Metrics:      metrics,
		Extends:      []string{"_base.yaml", "_generic-if.yaml"},
		Device:       deviceMeta{Vendor: "f5"},
		SysObjectIds: StringArray{"1.3.6.1.4.1.3375.2.1.3.4.*"},
		MetricTags:   []metricTagConfig{{Tag: "snmp_host", Index: 0x0, Column: symbolConfig{OID: "", Name: ""}, OID: "1.3.6.1.2.1.1.5.0", Name: "sysName"}},
	}}
}

func Test_getDefaultProfilesDefinitionFiles(t *testing.T) {
	setConfdPath()
	actualProfileConfig := getDefaultProfilesDefinitionFiles()

	confdPath := config.Datadog.GetString("confd_path")
	expectedProfileConfig := profilesConfig{
		"f5-big-ip": {
			filepath.Join(confdPath, "snmp.d", "profiles", "f5-big-ip.yaml"),
		},
	}

	assert.Equal(t, expectedProfileConfig, actualProfileConfig)
}

func Test_loadProfiles(t *testing.T) {
	setConfdPath()
	files := getDefaultProfilesDefinitionFiles()
	profiles, err := loadProfiles(files)
	assert.Nil(t, err)

	assert.Equal(t, mockProfilesDefinitions(), profiles)
}
