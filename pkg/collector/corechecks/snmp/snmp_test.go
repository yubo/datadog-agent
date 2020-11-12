// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2020 Datadog, Inc.

package snmp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBasicConfiguration(t *testing.T) {
	check := Check{}
	rawInstanceConfig := []byte(`
ip_address: 1.2.3.4
port: 1161
metrics:
- OID: 1.3.6.1.2.1.1.3.0
  name: sysUpTimeInstance
- OID: 1.3.6.1.2.1.2.1
  name: ifNumber
`)
	err := check.Configure(rawInstanceConfig, []byte(``), "test")

	assert.Nil(t, err)
	assert.Equal(t, "1.2.3.4", check.config.instance.IPAddress)
	assert.Equal(t, 1161, check.config.instance.Port)
	metrics := []metricsConfig{
		{OID: "1.3.6.1.2.1.1.3.0", Name: "sysUpTimeInstance"},
		{OID: "1.3.6.1.2.1.2.1", Name: "ifNumber"},
	}
	assert.Equal(t, metrics, check.config.instance.Metrics)
}
