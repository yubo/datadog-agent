// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2020 Datadog, Inc.

package snmp

import (
	"github.com/DataDog/datadog-agent/pkg/aggregator/mocksender"
	"github.com/soniah/gosnmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
)

type mockSession struct {
	mock.Mock
}

func (s *mockSession) Configure(config snmpConfig) {
}

func (s *mockSession) Connect() error {
	return nil
}

func (s *mockSession) Close() error {
	return nil
}

func (s *mockSession) Get(oids []string) (result *gosnmp.SnmpPacket, err error) {
	args := s.Mock.Called(oids)
	return args.Get(0).(*gosnmp.SnmpPacket), args.Error(1)
}

func TestBasicSample(t *testing.T) {
	session := &mockSession{}
	check := Check{session: session}
	// language=yaml
	rawInstanceConfig := []byte(`
ip_address: 1.2.3.4
metrics:
- OID: 1.3.6.1.2.1.1.3.0
  name: sysUpTimeInstance
- OID: 1.3.6.1.2.1.2.1
  name: ifNumber
`)

	err := check.Configure(rawInstanceConfig, []byte(``), "test")
	assert.Nil(t, err)

	sender := mocksender.NewMockSender(check.ID()) // required to initiate aggregator
	sender.On("Gauge", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()
	sender.On("Commit").Return()

	packet := gosnmp.SnmpPacket{
		Variables: []gosnmp.SnmpPDU{
			{
				Name:  "1.3.6.1.2.1.1.3.0",
				Type:  gosnmp.TimeTicks,
				Value: 20,
			},
			{
				Name:  "1.3.6.1.2.1.2.1",
				Type:  gosnmp.Integer,
				Value: 30,
			},
		},
	}

	session.On("Get", mock.Anything).Return(&packet, nil)

	err = check.Run()
	assert.Nil(t, err)

	sender.AssertCalled(t, "Gauge", "snmp.devices_monitored", float64(1), "", []string(nil))
	sender.AssertCalled(t, "Gauge", "snmp.sysUpTimeInstance", float64(20), "", []string(nil))
	sender.AssertCalled(t, "Gauge", "snmp.ifNumber", float64(30), "", []string(nil))
}
