// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2020 Datadog, Inc.

package snmp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfigurations(t *testing.T) {
	setConfdPath()

	check := Check{session: &snmpSession{}}
	// language=yaml
	rawInstanceConfig := []byte(`
ip_address: 1.2.3.4
port: 1161
timeout: 7
retries: 5
snmp_version: 2c
user: my-user
authProtocol: sha
authKey: my-AuthKey
privProtocol: aes
privKey: my-PrivKey
context_name: my-ContextName
metrics:
- symbol:
    OID: 1.3.6.1.2.1.2.1
    name: ifNumber
- OID: 1.3.6.1.2.1.2.2
  name: ifNumber2
- symbol:
    OID: 1.3.6.1.4.1.318.1.1.1.11.1.1.0
    name: upsBasicStateOutputState
  forced_type: flag_stream
  options:
    placement: 5
    metric_suffix: ReplaceBattery
- table:
    OID: 1.3.6.1.2.1.2.2
    name: ifTable
  symbols:
  - OID: 1.3.6.1.2.1.2.2.1.14
    name: ifInErrors
  - OID: 1.3.6.1.2.1.2.2.1.20
    name: ifOutErrors
  metric_tags:
  - tag: if_index
    index: 1
  - tag: if_desc
    column:
      OID: 1.3.6.1.2.1.2.2.1.2
      name: ifDescr
    index_transform:
      - start: 1
        end: 3
      - start: 4
        end: 6
  - index: 1
    tag: ipversion
    mapping:
      0: unknown
      1: ipv4
      2: ipv6
      3: ipv4z
      4: ipv6z
      16: dns
metric_tags:
  - OID: 1.2.3
    symbol: mySymbol
    tag: my_symbol
profile: f5-big-ip
`)
	// language=yaml
	rawInitConfig := []byte(`
profiles:
  f5-big-ip:
    definition_file: f5-big-ip.yaml
global_metrics:
- symbol:
    OID: 1.2.3.4
    name: aGlobalMetric
`)
	err := check.Configure(rawInstanceConfig, rawInitConfig, "test")

	assert.Nil(t, err)
	assert.Equal(t, "1.2.3.4", check.config.IPAddress)
	assert.Equal(t, uint16(1161), check.config.Port)
	assert.Equal(t, 7, check.config.Timeout)
	assert.Equal(t, 5, check.config.Retries)
	assert.Equal(t, "2c", check.config.SnmpVersion)
	assert.Equal(t, "my-user", check.config.User)
	assert.Equal(t, "sha", check.config.AuthProtocol)
	assert.Equal(t, "my-AuthKey", check.config.AuthKey)
	assert.Equal(t, "aes", check.config.PrivProtocol)
	assert.Equal(t, "my-PrivKey", check.config.PrivKey)
	assert.Equal(t, "my-ContextName", check.config.ContextName)
	metrics := []metricsConfig{
		{Symbol: symbolConfig{OID: "1.3.6.1.2.1.2.1", Name: "ifNumber"}},
		{Symbol: symbolConfig{OID: "1.3.6.1.2.1.2.2", Name: "ifNumber2"}},
		{Symbol: symbolConfig{OID: "1.3.6.1.4.1.318.1.1.1.11.1.1.0", Name: "upsBasicStateOutputState"}, ForcedType: "flag_stream", Options: metricsConfigOption{Placement: 5, MetricSuffix: "ReplaceBattery"}},
		{
			Table: symbolConfig{OID: "1.3.6.1.2.1.2.2", Name: "ifTable"},
			Symbols: []symbolConfig{
				{OID: "1.3.6.1.2.1.2.2.1.14", Name: "ifInErrors"},
				{OID: "1.3.6.1.2.1.2.2.1.20", Name: "ifOutErrors"},
			},
			MetricTags: []metricTagConfig{
				{Tag: "if_index", Index: 1},
				{Tag: "if_desc", Column: symbolConfig{OID: "1.3.6.1.2.1.2.2.1.2", Name: "ifDescr"},
					IndexTransform: []metricIndexTransform{
						{
							Start: 1,
							End:   3,
						},
						{
							Start: 4,
							End:   6,
						},
					},
				},
				{Tag: "ipversion", Index: 1, Mapping: map[string]string{
					"0":  "unknown",
					"1":  "ipv4",
					"2":  "ipv6",
					"3":  "ipv4z",
					"4":  "ipv6z",
					"16": "dns",
				}},
			},
		},
		{Symbol: symbolConfig{OID: "1.2.3.4", Name: "aGlobalMetric"}},
	}
	metrics = append(metrics, mockProfilesDefinitions()["f5-big-ip"].Metrics...)

	metricsTags := []metricTagConfig{
		{Tag: "my_symbol", OID: "1.2.3", Name: "mySymbol"},
		{Tag: "snmp_host", OID: "1.3.6.1.2.1.1.5.0", Name: "sysName"},
	}

	assert.Equal(t, metrics, check.config.Metrics)
	assert.Equal(t, metricsTags, check.config.MetricTags)
	assert.Equal(t, 1, len(check.config.Profiles))
}

func TestDefaultConfigurations(t *testing.T) {
	setConfdPath()

	check := Check{session: &snmpSession{}}
	// language=yaml
	rawInstanceConfig := []byte(`
ip_address: 1.2.3.4
`)
	// language=yaml
	rawInitConfig := []byte(``)
	err := check.Configure(rawInstanceConfig, rawInitConfig, "test")

	assert.Nil(t, err)
	assert.Equal(t, "1.2.3.4", check.config.IPAddress)
	assert.Equal(t, uint16(161), check.config.Port)
	assert.Equal(t, 2, check.config.Timeout)
	assert.Equal(t, 3, check.config.Retries)
	var metrics []metricsConfig

	var metricsTags []metricTagConfig

	assert.Equal(t, metrics, check.config.Metrics)
	assert.Equal(t, metricsTags, check.config.MetricTags)
	assert.Equal(t, 1, len(check.config.Profiles))
	assert.Equal(t, mockProfilesDefinitions()["f5-big-ip"].Metrics, check.config.Profiles["f5-big-ip"].Metrics)
}

func TestPortConfiguration(t *testing.T) {
	// TEST Default port
	check := Check{session: &snmpSession{}}
	// language=yaml
	rawInstanceConfig := []byte(`
ip_address: 1.2.3.4
`)
	err := check.Configure(rawInstanceConfig, []byte(``), "test")
	assert.Nil(t, err)
	assert.Equal(t, uint16(161), check.config.Port)

	// TEST Custom port
	check = Check{session: &snmpSession{}}
	// language=yaml
	rawInstanceConfig = []byte(`
ip_address: 1.2.3.4
port: 1234
`)
	err = check.Configure(rawInstanceConfig, []byte(``), "test")
	assert.Nil(t, err)
	assert.Equal(t, uint16(1234), check.config.Port)
}

func TestGlobalMetricsConfigurations(t *testing.T) {
	setConfdPath()

	check := Check{session: &snmpSession{}}
	// language=yaml
	rawInstanceConfig := []byte(`
ip_address: 1.2.3.4
metrics:
- symbol:
    OID: 1.3.6.1.2.1.2.1
    name: ifNumber
`)
	// language=yaml
	rawInitConfig := []byte(`
global_metrics:
- symbol:
    OID: 1.2.3.4
    name: aGlobalMetric
`)
	err := check.Configure(rawInstanceConfig, rawInitConfig, "test")
	assert.Nil(t, err)

	metrics := []metricsConfig{
		{Symbol: symbolConfig{OID: "1.3.6.1.2.1.2.1", Name: "ifNumber"}},
		{Symbol: symbolConfig{OID: "1.2.3.4", Name: "aGlobalMetric"}},
	}
	assert.Equal(t, metrics, check.config.Metrics)
}

func TestUseGlobalMetricsFalse(t *testing.T) {
	setConfdPath()

	check := Check{session: &snmpSession{}}
	// language=yaml
	rawInstanceConfig := []byte(`
ip_address: 1.2.3.4
metrics:
- symbol:
    OID: 1.3.6.1.2.1.2.1
    name: aInstanceMetric
use_global_metrics: false
`)
	// language=yaml
	rawInitConfig := []byte(`
global_metrics:
- symbol:
    OID: 1.2.3.4
    name: aGlobalMetric
`)
	err := check.Configure(rawInstanceConfig, rawInitConfig, "test")
	assert.Nil(t, err)

	metrics := []metricsConfig{
		{Symbol: symbolConfig{OID: "1.3.6.1.2.1.2.1", Name: "aInstanceMetric"}},
	}
	assert.Equal(t, metrics, check.config.Metrics)
}

func Test_oidConfig_hasOids(t *testing.T) {
	tests := []struct {
		name            string
		scalarOids      []string
		columnOids      []string
		expectedHasOids bool
	}{
		{
			"has scalar oids",
			[]string{"1.2.3"},
			[]string{},
			true,
		},
		{
			"has scalar and column oids",
			[]string{"1.2.3"},
			[]string{"1.2.4"},
			true,
		},
		{
			"has no oids",
			[]string{},
			[]string{},
			false,
		},
		{
			"has no oids nil",
			nil,
			nil,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oc := &oidConfig{
				scalarOids: tt.scalarOids,
				columnOids: tt.columnOids,
			}
			hasOids := oc.hasOids()
			assert.Equal(t, tt.expectedHasOids, hasOids)
		})
	}
}
