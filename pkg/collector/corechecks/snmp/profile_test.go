package snmp

import (
	"fmt"
	"github.com/DataDog/datadog-agent/pkg/config"
	assert "github.com/stretchr/testify/require"
	"path/filepath"
	"testing"
)

func mockProfilesDefinitions() profileDefinitionMap {
	metrics := []metricsConfig{
		{Symbol: symbolConfig{OID: "1.3.6.1.4.1.3375.2.1.1.2.1.44.0", Name: "sysStatMemoryTotal"}, ForcedType: "gauge"},
		{Symbol: symbolConfig{OID: "1.3.6.1.4.1.3375.2.1.1.2.1.44.999", Name: "oldSyntax"}},
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

	for _, profile := range profiles {
		normalizeMetrics(profile.Metrics)
	}

	assert.Equal(t, mockProfilesDefinitions(), profiles)
}

func Test_getMostSpecificOid(t *testing.T) {
	tests := []struct {
		name           string
		oids           []string
		expectedOid    string
		expectedErrror error
	}{
		{
			"one",
			[]string{"1.2.3.4"},
			"1.2.3.4",
			nil,
		},
		{
			"error on empty oids",
			[]string{},
			"",
			fmt.Errorf("cannot get most specific oid from empty list of oids"),
		},
		{
			"error on parsing",
			[]string{"a.1.2.3"},
			"",
			fmt.Errorf("error parsing part `a` for pattern `a.1.2.3`: strconv.Atoi: parsing \"a\": invalid syntax"),
		},
		{
			"most lengthy",
			[]string{"1.3.4", "1.3.4.1.2"},
			"1.3.4.1.2",
			nil,
		},
		{
			"wild card 1",
			[]string{"1.3.4.*", "1.3.4.1"},
			"1.3.4.1",
			nil,
		},
		{
			"wild card 2",
			[]string{"1.3.4.1", "1.3.4.*"},
			"1.3.4.1",
			nil,
		},
		{
			"sample oids",
			[]string{"1.3.6.1.4.1.3375.2.1.3.4.43", "1.3.6.1.4.1.8072.3.2.10"},
			"1.3.6.1.4.1.3375.2.1.3.4.43",
			nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oid, err := getMostSpecificOid(tt.oids)
			assert.Equal(t, tt.expectedErrror, err)
			assert.Equal(t, tt.expectedOid, oid)
		})
	}
}
