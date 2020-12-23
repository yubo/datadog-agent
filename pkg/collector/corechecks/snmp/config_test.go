// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2020 Datadog, Inc.

package snmp

import (
	"github.com/DataDog/datadog-agent/pkg/config"
	"github.com/soniah/gosnmp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBasicConfiguration(t *testing.T) {
	config.Datadog.Set("confd_path", "./test/conf.d")

	check := Check{session: &snmpSession{}}
	// language=yaml
	rawInstanceConfig := []byte(`
ip_address: 1.2.3.4
port: 1161
timeout: 7
retries: 5
metrics:
- symbol:
    OID: 1.3.6.1.2.1.2.1
    name: ifNumber
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
metric_tags:
  - OID: 1.2.3
    symbol: mySymbol
    tag: my_symbol
`)
	// language=yaml
	rawInitConfig := []byte(`
profiles:
  f5-big-ip:
    definition_file: f5-big-ip.yaml
profile: f5-big-ip
`)
	err := check.Configure(rawInstanceConfig, rawInitConfig, "test")

	assert.Nil(t, err)
	assert.Equal(t, "1.2.3.4", check.config.IPAddress)
	assert.Equal(t, uint16(1161), check.config.Port)
	assert.Equal(t, 7, check.config.Timeout)
	assert.Equal(t, 5, check.config.Retries)
	metrics := []metricsConfig{
		{Symbol: symbolConfig{OID: "1.3.6.1.2.1.2.1", Name: "ifNumber"}},
		{
			Table: symbolConfig{OID: "1.3.6.1.2.1.2.2", Name: "ifTable"},
			Symbols: []symbolConfig{
				{OID: "1.3.6.1.2.1.2.2.1.14", Name: "ifInErrors"},
				{OID: "1.3.6.1.2.1.2.2.1.20", Name: "ifOutErrors"},
			},
			MetricTags: []metricTagConfig{
				{Tag: "if_index", Index: 1},
				{Tag: "if_desc", Column: symbolConfig{OID: "1.3.6.1.2.1.2.2.1.2", Name: "ifDescr"}},
			},
		},
		{Symbol: symbolConfig{OID: "1.3.6.1.2.1.1.3.0", Name: "sysUpTimeInstance"}},
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
	}

	metricsTags := []metricTagConfig{
		{Tag: "my_symbol", OID: "1.2.3", Name: "mySymbol"},
		{Tag: "snmp_host", OID: "1.3.6.1.2.1.1.5.0", Name: "sysName"},
	}

	assert.Equal(t, metrics, check.config.Metrics)
	assert.Equal(t, metricsTags, check.config.MetricTags)
	assert.Equal(t, 1, len(check.config.Profiles))
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

func TestVersionConfiguration(t *testing.T) {
	// TEST Empty case
	check := Check{session: &snmpSession{}}
	// language=yaml
	rawInstanceConfig := []byte(`
ip_address: 1.2.3.4
`)
	err := check.Configure(rawInstanceConfig, []byte(``), "test")
	assert.Nil(t, err)
	assert.Equal(t, gosnmp.Version2c, check.config.SnmpVersion)

	// TEST Valid versions
	cases := []struct {
		rawInstanceConfig []byte
		expectedVersion   gosnmp.SnmpVersion
	}{
		// language=yaml
		{[]byte(`
ip_address: 1.2.3.4
snmp_version: 2
`), gosnmp.Version2c},
		// language=yaml
		{[]byte(`
ip_address: 1.2.3.4
snmp_version: 1
`), gosnmp.Version1},
		// language=yaml
		{[]byte(`
ip_address: 1.2.3.4
snmp_version: 2
`), gosnmp.Version2c},
		// language=yaml
		{[]byte(`
ip_address: 1.2.3.4
snmp_version: 2c
`), gosnmp.Version2c},
		// language=yaml
		{[]byte(`
ip_address: 1.2.3.4
snmp_version: 3
`), gosnmp.Version3},
	}
	for _, tc := range cases {
		check = Check{session: &snmpSession{}}
		err = check.Configure(tc.rawInstanceConfig, []byte(``), "test")
		assert.Nil(t, err)
		assert.Equal(t, tc.expectedVersion, check.config.SnmpVersion)
	}

	// TEST Invalid version
	check = Check{session: &snmpSession{}}
	// language=yaml
	rawInstanceConfig = []byte(`
ip_address: 1.2.3.4
snmp_version: 4
`)
	err = check.Configure(rawInstanceConfig, []byte(``), "test")
	assert.Error(t, err, "invalid snmp version `4`. Valid versions are: 1, 2, 2c, 3")
}
