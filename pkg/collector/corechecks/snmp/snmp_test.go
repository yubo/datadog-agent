// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2020 Datadog, Inc.

package snmp

import (
	"github.com/DataDog/datadog-agent/pkg/aggregator"
	"github.com/DataDog/datadog-agent/pkg/aggregator/mocksender"
	"github.com/DataDog/datadog-agent/pkg/collector/check"
	"github.com/DataDog/datadog-agent/pkg/config"
	"github.com/soniah/gosnmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"path/filepath"
	"testing"
	"time"
)

type mockSession struct {
	mock.Mock
}

func (s *mockSession) Configure(config snmpConfig) error {
	return nil
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

func setConfdPath() {
	file, _ := filepath.Abs("./test/conf.d")
	config.Datadog.Set("confd_path", file)
}

func TestBasicSample(t *testing.T) {
	setConfdPath()
	session := &mockSession{}
	check := Check{session: session}
	aggregator.InitAggregatorWithFlushInterval(nil, "", 1*time.Hour)

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
metric_tags:
  - OID: 1.3.6.1.2.1.1.5.0
    symbol: sysName
    tag: snmp_host
tags:
  - "mytag:foo"
`)

	err := check.Configure(rawInstanceConfig, []byte(``), "test")
	assert.Nil(t, err)

	sender := mocksender.NewMockSender(check.ID()) // required to initiate aggregator
	sender.On("Gauge", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()
	sender.On("MonotonicCount", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()
	sender.On("ServiceCheck", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()
	sender.On("Commit").Return()

	packet := gosnmp.SnmpPacket{
		Variables: []gosnmp.SnmpPDU{
			{
				Name:  "1.3.6.1.2.1.1.3.0",
				Type:  gosnmp.TimeTicks,
				Value: 20,
			},
			{
				Name:  "1.3.6.1.2.1.1.5.0",
				Type:  gosnmp.OctetString,
				Value: []byte("foo_sys_name"),
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

	snmpTags := []string{"snmp_device:1.2.3.4", "loader:core"}
	snmpGlobalTags := []string{"snmp_host:foo_sys_name"}
	snmpGlobalTags = append(snmpGlobalTags, snmpTags...)

	var row1Tags, row2Tags []string
	row1Tags = append(row1Tags, snmpGlobalTags...)
	row1Tags = append(row1Tags, "if_index:1", "if_desc:desc1")
	row2Tags = append(row2Tags, snmpGlobalTags...)
	row2Tags = append(row2Tags, "if_index:2", "if_desc:desc2")

	sender.AssertMetric(t, "Gauge", "snmp.devices_monitored", float64(1), "", snmpGlobalTags)
	sender.AssertMetric(t, "Gauge", "snmp.sysUpTimeInstance", float64(20), "", snmpGlobalTags)
	sender.AssertMetric(t, "Gauge", "snmp.ifNumber", float64(30), "", snmpGlobalTags)
	sender.AssertMetric(t, "Gauge", "snmp.ifInErrors", float64(141), "", row1Tags)
	sender.AssertMetric(t, "Gauge", "snmp.ifInErrors", float64(142), "", row2Tags)
	sender.AssertMetric(t, "Gauge", "snmp.ifOutErrors", float64(201), "", row1Tags)
	sender.AssertMetric(t, "Gauge", "snmp.ifOutErrors", float64(202), "", row2Tags)

	sender.AssertMetricTaggedWith(t, "MonotonicCount", "snmp.check_interval", snmpTags)
	sender.AssertMetricTaggedWith(t, "Gauge", "snmp.check_duration", snmpGlobalTags)
	sender.AssertMetricTaggedWith(t, "Gauge", "snmp.submitted_metrics", snmpGlobalTags)
}

func TestSupportedMetricTypes(t *testing.T) {
	setConfdPath()
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
	sender.On("MonotonicCount", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()
	sender.On("Rate", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()
	sender.On("ServiceCheck", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()
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

	tags := []string{"loader:core", "snmp_device:1.2.3.4"}
	sender.AssertMetric(t, "Gauge", "snmp.devices_monitored", float64(1), "", tags)
	sender.AssertMetric(t, "Gauge", "snmp.sysUpTimeInstance", float64(20), "", tags)
	sender.AssertMetric(t, "Gauge", "snmp.SomeGaugeMetric", float64(30), "", tags)
	sender.AssertMetric(t, "Rate", "snmp.SomeCounter32Metric", float64(40), "", tags)
	sender.AssertMetric(t, "Rate", "snmp.SomeCounter64Metric", float64(50), "", tags)
}

func TestProfile(t *testing.T) {
	setConfdPath()
	session := &mockSession{}
	check := Check{session: session}
	// language=yaml
	rawInstanceConfig := []byte(`
ip_address: 1.2.3.4
profile: f5-big-ip
`)
	// language=yaml
	rawInitConfig := []byte(`
profiles:
  f5-big-ip:
    definition_file: f5-big-ip.yaml
`)

	err := check.Configure(rawInstanceConfig, rawInitConfig, "test")
	assert.Nil(t, err)

	sender := mocksender.NewMockSender(check.ID()) // required to initiate aggregator
	sender.On("Gauge", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()
	sender.On("MonotonicCount", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()
	sender.On("ServiceCheck", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()
	sender.On("Commit").Return()

	packet := gosnmp.SnmpPacket{
		Variables: []gosnmp.SnmpPDU{
			{
				Name:  "1.3.6.1.2.1.1.5.0",
				Type:  gosnmp.OctetString,
				Value: []byte("foo_sys_name"),
			},
			{
				Name:  "1.3.6.1.4.1.3375.2.1.1.2.1.44.0",
				Type:  gosnmp.Integer,
				Value: 30,
			},
			{
				Name:  "1.3.6.1.2.1.1.3.0",
				Type:  gosnmp.TimeTicks,
				Value: 20,
			},
		},
	}

	bulkPacket := gosnmp.SnmpPacket{
		Variables: []gosnmp.SnmpPDU{
			{
				Name:  "1.3.6.1.2.1.2.2.1.13.1",
				Type:  gosnmp.Integer,
				Value: 131,
			},
			{
				Name:  "1.3.6.1.2.1.2.2.1.14.1",
				Type:  gosnmp.Integer,
				Value: 141,
			},
			{
				Name:  "1.3.6.1.2.1.31.1.1.1.1.1",
				Type:  gosnmp.OctetString,
				Value: []byte("nameRow1"),
			},
			{
				Name:  "1.3.6.1.2.1.31.1.1.1.18.1",
				Type:  gosnmp.OctetString,
				Value: []byte("descRow1"),
			},
			{
				Name:  "1.3.6.1.2.1.2.2.1.13.2",
				Type:  gosnmp.Integer,
				Value: 132,
			},
			{
				Name:  "1.3.6.1.2.1.2.2.1.14.2",
				Type:  gosnmp.Integer,
				Value: 142,
			},
			{
				Name:  "1.3.6.1.2.1.31.1.1.1.1.2",
				Type:  gosnmp.OctetString,
				Value: []byte("nameRow2"),
			},
			{
				Name:  "1.3.6.1.2.1.31.1.1.1.18.2",
				Type:  gosnmp.OctetString,
				Value: []byte("descRow2"),
			},
			{
				Name:  "9", // exit table
				Type:  gosnmp.Integer,
				Value: 999,
			},
			{
				Name:  "9", // exit table
				Type:  gosnmp.Integer,
				Value: 999,
			},
			{
				Name:  "9", // exit table
				Type:  gosnmp.Integer,
				Value: 999,
			},
			{
				Name:  "9", // exit table
				Type:  gosnmp.Integer,
				Value: 999,
			},
		},
	}

	session.On("Get", []string{"1.3.6.1.4.1.3375.2.1.1.2.1.44.0", "1.3.6.1.4.1.3375.2.1.1.2.1.44.999", "1.2.3.4.5", "1.3.6.1.2.1.1.5.0", "1.3.6.1.2.1.1.3.0"}).Return(&packet, nil)
	session.On("GetBulk", []string{"1.3.6.1.2.1.2.2.1.13", "1.3.6.1.2.1.2.2.1.14", "1.3.6.1.2.1.31.1.1.1.1", "1.3.6.1.2.1.31.1.1.1.18"}).Return(&bulkPacket, nil)

	err = check.Run()
	assert.Nil(t, err)

	snmpTags := []string{"snmp_device:1.2.3.4", "loader:core", "snmp_profile:f5-big-ip", "device_vendor:f5", "snmp_host:foo_sys_name"}

	var row1Tags, row2Tags []string
	row1Tags = append(row1Tags, snmpTags...)
	row1Tags = append(row1Tags, "interface:nameRow1", "interface_alias:descRow1")
	row2Tags = append(row2Tags, snmpTags...)
	row2Tags = append(row2Tags, "interface:nameRow2", "interface_alias:descRow2")

	sender.AssertMetric(t, "Gauge", "snmp.devices_monitored", float64(1), "", snmpTags)
	sender.AssertMetric(t, "Gauge", "snmp.sysUpTimeInstance", float64(20), "", snmpTags)
	sender.AssertMetric(t, "MonotonicCount", "snmp.ifInErrors", float64(141), "", row1Tags)
	sender.AssertMetric(t, "MonotonicCount", "snmp.ifInErrors", float64(142), "", row2Tags)
	sender.AssertMetric(t, "MonotonicCount", "snmp.ifInDiscards", float64(131), "", row1Tags)
	sender.AssertMetric(t, "MonotonicCount", "snmp.ifInDiscards", float64(132), "", row2Tags)
	sender.AssertMetric(t, "Gauge", "snmp.sysStatMemoryTotal", float64(30), "", snmpTags)
}

func TestProfileWithSysObjectIdDetection(t *testing.T) {
	setConfdPath()
	session := &mockSession{}
	check := Check{session: session}
	// language=yaml
	rawInstanceConfig := []byte(`
ip_address: 1.2.3.4
`)
	// language=yaml
	rawInitConfig := []byte(`
profiles:
  f5-big-ip:
    definition_file: f5-big-ip.yaml
`)

	err := check.Configure(rawInstanceConfig, rawInitConfig, "test")
	assert.Nil(t, err)

	sender := mocksender.NewMockSender(check.ID()) // required to initiate aggregator
	sender.On("Gauge", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()
	sender.On("MonotonicCount", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()
	sender.On("ServiceCheck", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()
	sender.On("Commit").Return()

	sysObjectIDPacket := gosnmp.SnmpPacket{
		Variables: []gosnmp.SnmpPDU{
			{
				Name:  "1.3.6.1.2.1.1.2.0",
				Type:  gosnmp.ObjectIdentifier,
				Value: "1.3.6.1.4.1.3375.2.1.3.4.1",
			},
		},
	}

	packet := gosnmp.SnmpPacket{
		Variables: []gosnmp.SnmpPDU{
			{
				Name:  "1.3.6.1.2.1.1.3.0",
				Type:  gosnmp.TimeTicks,
				Value: 20,
			},
			{
				Name:  "1.3.6.1.2.1.1.5.0",
				Type:  gosnmp.OctetString,
				Value: []byte("foo_sys_name"),
			},
			{
				Name:  "1.3.6.1.4.1.3375.2.1.1.2.1.44.0",
				Type:  gosnmp.Integer,
				Value: 30,
			},
		},
	}

	bulkPacket := gosnmp.SnmpPacket{
		Variables: []gosnmp.SnmpPDU{
			{
				Name:  "1.3.6.1.2.1.2.2.1.13.1",
				Type:  gosnmp.Integer,
				Value: 131,
			},
			{
				Name:  "1.3.6.1.2.1.2.2.1.14.1",
				Type:  gosnmp.Integer,
				Value: 141,
			},
			{
				Name:  "1.3.6.1.2.1.31.1.1.1.1.1",
				Type:  gosnmp.OctetString,
				Value: []byte("nameRow1"),
			},
			{
				Name:  "1.3.6.1.2.1.31.1.1.1.18.1",
				Type:  gosnmp.OctetString,
				Value: []byte("descRow1"),
			},
			{
				Name:  "1.3.6.1.2.1.2.2.1.13.2",
				Type:  gosnmp.Integer,
				Value: 132,
			},
			{
				Name:  "1.3.6.1.2.1.2.2.1.14.2",
				Type:  gosnmp.Integer,
				Value: 142,
			},
			{
				Name:  "1.3.6.1.2.1.31.1.1.1.1.2",
				Type:  gosnmp.OctetString,
				Value: []byte("nameRow2"),
			},
			{
				Name:  "1.3.6.1.2.1.31.1.1.1.18.2",
				Type:  gosnmp.OctetString,
				Value: []byte("descRow2"),
			},
			{
				Name:  "9", // exit table
				Type:  gosnmp.Integer,
				Value: 999,
			},
			{
				Name:  "9", // exit table
				Type:  gosnmp.Integer,
				Value: 999,
			},
			{
				Name:  "9", // exit table
				Type:  gosnmp.Integer,
				Value: 999,
			},
			{
				Name:  "9", // exit table
				Type:  gosnmp.Integer,
				Value: 999,
			},
		},
	}

	session.On("Get", []string{"1.3.6.1.2.1.1.2.0"}).Return(&sysObjectIDPacket, nil)
	session.On("Get", []string{"1.3.6.1.4.1.3375.2.1.1.2.1.44.0", "1.3.6.1.4.1.3375.2.1.1.2.1.44.999", "1.2.3.4.5", "1.3.6.1.2.1.1.5.0", "1.3.6.1.2.1.1.3.0"}).Return(&packet, nil)
	session.On("GetBulk", []string{"1.3.6.1.2.1.2.2.1.13", "1.3.6.1.2.1.2.2.1.14", "1.3.6.1.2.1.31.1.1.1.1", "1.3.6.1.2.1.31.1.1.1.18"}).Return(&bulkPacket, nil)

	err = check.Run()
	assert.Nil(t, err)

	snmpTags := []string{"snmp_device:1.2.3.4", "loader:core", "snmp_profile:f5-big-ip", "device_vendor:f5", "snmp_host:foo_sys_name"}

	var row1Tags, row2Tags []string
	row1Tags = append(row1Tags, snmpTags...)
	row1Tags = append(row1Tags, "interface:nameRow1", "interface_alias:descRow1")
	row2Tags = append(row2Tags, snmpTags...)
	row2Tags = append(row2Tags, "interface:nameRow2", "interface_alias:descRow2")

	sender.AssertMetric(t, "Gauge", "snmp.devices_monitored", float64(1), "", snmpTags)
	sender.AssertMetric(t, "Gauge", "snmp.sysUpTimeInstance", float64(20), "", snmpTags)
	sender.AssertMetric(t, "MonotonicCount", "snmp.ifInErrors", float64(141), "", row1Tags)
	sender.AssertMetric(t, "MonotonicCount", "snmp.ifInErrors", float64(142), "", row2Tags)
	sender.AssertMetric(t, "MonotonicCount", "snmp.ifInDiscards", float64(131), "", row1Tags)
	sender.AssertMetric(t, "MonotonicCount", "snmp.ifInDiscards", float64(132), "", row2Tags)
	sender.AssertMetric(t, "Gauge", "snmp.sysStatMemoryTotal", float64(30), "", snmpTags)
}

func TestCheckID(t *testing.T) {
	setConfdPath()
	check1 := snmpFactory()
	check2 := snmpFactory()
	// language=yaml
	rawInstanceConfig1 := []byte(`
ip_address: 1.1.1.1
`)
	// language=yaml
	rawInstanceConfig2 := []byte(`
ip_address: 2.2.2.2
`)

	err := check1.Configure(rawInstanceConfig1, []byte(``), "test")
	assert.Nil(t, err)

	err = check2.Configure(rawInstanceConfig2, []byte(``), "test")
	assert.Nil(t, err)

	assert.Equal(t, check.ID("snmp:efc9e7e750047b05"), check1.ID())
	assert.Equal(t, check.ID("snmp:f22c3d8b3858f07d"), check2.ID())
	assert.NotEqual(t, check1.ID(), check2.ID())
}
