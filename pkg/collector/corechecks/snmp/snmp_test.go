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

func (s *mockSession) GetBulk(oids []string) (result *gosnmp.SnmpPacket, err error) {
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

  # TODO: Create separate test for index metric_tags
  metric_tags:
  - tag: if_index
    index: 1
  - tag: if_desc
    column:
      OID: 1.3.6.1.2.1.2.2.1.2
      name: ifDescr
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

	bulkPacket := gosnmp.SnmpPacket{
		Variables: []gosnmp.SnmpPDU{
			{
				Name:  "1.3.6.1.2.1.2.2.1.14.1",
				Type:  gosnmp.Integer,
				Value: 141,
			},
			{
				Name:  "1.3.6.1.2.1.2.2.1.2.1",
				Type:  gosnmp.OctetString,
				Value: []byte("desc1"),
			},
			{
				Name:  "1.3.6.1.2.1.2.2.1.20.1",
				Type:  gosnmp.Integer,
				Value: 201,
			},
			{
				Name:  "1.3.6.1.2.1.2.2.1.14.2",
				Type:  gosnmp.Integer,
				Value: 142,
			},
			{
				Name:  "1.3.6.1.2.1.2.2.1.2.2",
				Type:  gosnmp.OctetString,
				Value: []byte("desc2"),
			},
			{
				Name:  "1.3.6.1.2.1.2.2.1.20.2",
				Type:  gosnmp.Integer,
				Value: 202,
			},
		},
	}

	bulkPacket2 := gosnmp.SnmpPacket{
		Variables: []gosnmp.SnmpPDU{
			{
				Name:  "1.3.6.1.2.1.2.2.1.15.1",
				Type:  gosnmp.Integer,
				Value: 141,
			},
			{
				Name:  "1.3.6.1.2.1.2.2.1.3.2",
				Type:  gosnmp.OctetString,
				Value: []byte("none"),
			},
			{
				Name:  "1.3.6.1.2.1.2.2.1.21.1",
				Type:  gosnmp.Integer,
				Value: 201,
			},
		},
	}

	session.On("Get", mock.Anything).Return(&packet, nil)
	session.On("GetBulk", []string{"1.3.6.1.2.1.2.2.1.14", "1.3.6.1.2.1.2.2.1.2", "1.3.6.1.2.1.2.2.1.20"}).Return(&bulkPacket, nil)
	session.On("GetBulk", []string{"1.3.6.1.2.1.2.2.1.14.2", "1.3.6.1.2.1.2.2.1.2.2", "1.3.6.1.2.1.2.2.1.20.2"}).Return(&bulkPacket2, nil)

	err = check.Run()
	assert.Nil(t, err)

	tags := []string{"snmp_device:1.2.3.4"}
	sender.AssertCalled(t, "Gauge", "snmp.devices_monitored", float64(1), "", tags)
	sender.AssertCalled(t, "Gauge", "snmp.sysUpTimeInstance", float64(20), "", tags)
	sender.AssertCalled(t, "Gauge", "snmp.ifNumber", float64(30), "", tags)
	sender.AssertCalled(t, "Gauge", "snmp.ifInErrors", float64(141), "", append(tags, "if_index:1", "if_desc:desc1"))
	sender.AssertCalled(t, "Gauge", "snmp.ifInErrors", float64(142), "", append(tags, "if_index:2", "if_desc:desc2"))
	sender.AssertCalled(t, "Gauge", "snmp.ifOutErrors", float64(201), "", append(tags, "if_index:1", "if_desc:desc1"))
	sender.AssertCalled(t, "Gauge", "snmp.ifOutErrors", float64(202), "", append(tags, "if_index:2", "if_desc:desc2"))
}

func TestSupportedMetricTypes(t *testing.T) {
	session := &mockSession{}
	check := Check{session: session}
	// language=yaml
	rawInstanceConfig := []byte(`
ip_address: 1.2.3.4
metrics:
- symbol:
    OID: 1.2.3.4.5.0
    name: SomeGaugeMetric
- symbol:
    OID: 1.2.3.4.5.1
    name: SomeCounter32Metric
- symbol:
    OID: 1.2.3.4.5.2
    name: SomeCounter64Metric
`)

	err := check.Configure(rawInstanceConfig, []byte(``), "test")
	assert.Nil(t, err)

	sender := mocksender.NewMockSender(check.ID()) // required to initiate aggregator
	sender.On("Gauge", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()
	sender.On("Rate", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()
	sender.On("Commit").Return()

	packet := gosnmp.SnmpPacket{
		Variables: []gosnmp.SnmpPDU{
			{
				Name:  "1.3.6.1.2.1.1.3.0",
				Type:  gosnmp.TimeTicks,
				Value: 20,
			},
			{
				Name:  "1.2.3.4.5.0",
				Type:  gosnmp.Integer,
				Value: 30,
			},
			{
				Name:  "1.2.3.4.5.1",
				Type:  gosnmp.Counter32,
				Value: 40,
			},
			{
				Name:  "1.2.3.4.5.2",
				Type:  gosnmp.Counter64,
				Value: 50,
			},
		},
	}

	session.On("Get", mock.Anything).Return(&packet, nil)

	err = check.Run()
	assert.Nil(t, err)

	tags := []string{"snmp_device:1.2.3.4"}
	sender.AssertCalled(t, "Gauge", "snmp.devices_monitored", float64(1), "", tags)
	sender.AssertCalled(t, "Gauge", "snmp.sysUpTimeInstance", float64(20), "", tags)
	sender.AssertCalled(t, "Gauge", "snmp.SomeGaugeMetric", float64(30), "", tags)
	sender.AssertCalled(t, "Rate", "snmp.SomeCounter32Metric", float64(40), "", tags)
	sender.AssertCalled(t, "Rate", "snmp.SomeCounter64Metric", float64(50), "", tags)
}
