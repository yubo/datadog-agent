package snmp

import (
	"fmt"
	"github.com/DataDog/datadog-agent/pkg/metrics"
	"github.com/soniah/gosnmp"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_getValueFromPDU(t *testing.T) {
	tests := []struct {
		caseName          string
		pduVariable       gosnmp.SnmpPDU
		expectedName      string
		expectedSnmpValue snmpValue
		expectedErr       error
	}{
		{
			"Name",
			gosnmp.SnmpPDU{
				Name:  ".1.2.3",
				Type:  gosnmp.Integer,
				Value: 141,
			},
			"1.2.3",
			snmpValue{val: float64(141)},
			nil,
		},
		{
			"Integer",
			gosnmp.SnmpPDU{
				Name:  ".1.2.3",
				Type:  gosnmp.Integer,
				Value: 141,
			},
			"1.2.3",
			snmpValue{val: float64(141)},
			nil,
		},
		{
			"OctetString",
			gosnmp.SnmpPDU{
				Name:  ".1.2.3",
				Type:  gosnmp.OctetString,
				Value: []byte(`myVal`),
			},
			"1.2.3",
			snmpValue{val: "myVal"},
			nil,
		},
		{
			"OctetString hexify",
			gosnmp.SnmpPDU{
				Name:  ".1.2.3",
				Type:  gosnmp.OctetString,
				Value: []uint8{0x0, 0x24, 0x9b, 0x35, 0x3, 0xf6},
			},
			"1.2.3",
			snmpValue{val: "0x00249b3503f6"},
			nil,
		},
		{
			"BitString",
			gosnmp.SnmpPDU{
				Name:  ".1.2.3",
				Type:  gosnmp.BitString,
				Value: []byte(`myVal`),
			},
			"1.2.3",
			snmpValue{val: "myVal"},
			nil,
		},
		{
			"ObjectIdentifier",
			gosnmp.SnmpPDU{
				Name:  ".1.2.3",
				Type:  gosnmp.ObjectIdentifier,
				Value: "1.2.2",
			},
			"1.2.3",
			snmpValue{val: "1.2.2"},
			nil,
		},
		{
			"ObjectIdentifier need trim",
			gosnmp.SnmpPDU{
				Name:  ".1.2.3",
				Type:  gosnmp.ObjectIdentifier,
				Value: ".1.2.2",
			},
			"1.2.3",
			snmpValue{val: "1.2.2"},
			nil,
		},
		{
			"IPAddress",
			gosnmp.SnmpPDU{
				Name:  ".1.2.3",
				Type:  gosnmp.IPAddress,
				Value: "1.2.3.4",
			},
			"1.2.3",
			snmpValue{val: "1.2.3.4"},
			nil,
		},
		{
			"IPAddress invalid value",
			gosnmp.SnmpPDU{
				Name:  ".1.2.3",
				Type:  gosnmp.IPAddress,
				Value: nil,
			},
			"1.2.3",
			snmpValue{},
			fmt.Errorf("oid .1.2.3: invalid IP Address with value <nil>"),
		},
		{
			"Null",
			gosnmp.SnmpPDU{
				Name:  ".1.2.3",
				Type:  gosnmp.Null,
				Value: nil,
			},
			"1.2.3",
			snmpValue{},
			fmt.Errorf("oid .1.2.3: invalid type: Null"),
		},
		{
			"Counter32",
			gosnmp.SnmpPDU{
				Name:  ".1.2.3",
				Type:  gosnmp.Counter32,
				Value: uint(10),
			},
			"1.2.3",
			snmpValue{submissionType: metrics.RateType, val: float64(10)},
			nil,
		},
		{
			"Gauge32",
			gosnmp.SnmpPDU{
				Name:  ".1.2.3",
				Type:  gosnmp.Gauge32,
				Value: uint(10),
			},
			"1.2.3",
			snmpValue{val: float64(10)},
			nil,
		},
		{
			"TimeTicks",
			gosnmp.SnmpPDU{
				Name:  ".1.2.3",
				Type:  gosnmp.TimeTicks,
				Value: uint32(10),
			},
			"1.2.3",
			snmpValue{val: float64(10)},
			nil,
		},
		{
			"Counter64",
			gosnmp.SnmpPDU{
				Name:  ".1.2.3",
				Type:  gosnmp.Counter64,
				Value: uint64(10),
			},
			"1.2.3",
			snmpValue{submissionType: metrics.RateType, val: float64(10)},
			nil,
		},
		{
			"Uinteger32",
			gosnmp.SnmpPDU{
				Name:  ".1.2.3",
				Type:  gosnmp.Uinteger32,
				Value: uint32(10),
			},
			"1.2.3",
			snmpValue{val: float64(10)},
			nil,
		},
		{
			"OpaqueFloat",
			gosnmp.SnmpPDU{
				Name:  ".1.2.3",
				Type:  gosnmp.OpaqueFloat,
				Value: float32(10),
			},
			"1.2.3",
			snmpValue{val: float64(10)},
			nil,
		},
		{
			"OpaqueDouble",
			gosnmp.SnmpPDU{
				Name:  ".1.2.3",
				Type:  gosnmp.OpaqueDouble,
				Value: float64(10),
			},
			"1.2.3",
			snmpValue{val: float64(10)},
			nil,
		},
		{
			"NoSuchObject",
			gosnmp.SnmpPDU{
				Name:  ".1.2.3",
				Type:  gosnmp.NoSuchObject,
				Value: nil,
			},
			"1.2.3",
			snmpValue{},
			fmt.Errorf("oid .1.2.3: invalid type: NoSuchObject"),
		},
		{
			"NoSuchInstance",
			gosnmp.SnmpPDU{
				Name:  ".1.2.3",
				Type:  gosnmp.NoSuchInstance,
				Value: nil,
			},
			"1.2.3",
			snmpValue{},
			fmt.Errorf("oid .1.2.3: invalid type: NoSuchInstance"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.caseName, func(t *testing.T) {
			name, value, err := getValueFromPDU(tt.pduVariable)
			assert.Equal(t, name, tt.expectedName)
			assert.Equal(t, value, tt.expectedSnmpValue)
			assert.Equal(t, err, tt.expectedErr)
		})
	}
}

func Test_resultToColumnValues(t *testing.T) {
	tests := []struct {
		name                string
		columnOids          []string
		snmpPacket          *gosnmp.SnmpPacket
		expectedValues      columnResultValuesType
		expectedNextOidsMap map[string]string
	}{
		{
			"simple nominal case",
			[]string{"1.3.6.1.2.1.2.2.1.14", "1.3.6.1.2.1.2.2.1.2", "1.3.6.1.2.1.2.2.1.20"},
			&gosnmp.SnmpPacket{
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
			},
			columnResultValuesType{
				"1.3.6.1.2.1.2.2.1.14": {
					"1": snmpValue{
						val: float64(141),
					},
					"2": snmpValue{
						val: float64(142),
					},
				},
				"1.3.6.1.2.1.2.2.1.2": {
					"1": snmpValue{
						val: "desc1",
					},
					"2": snmpValue{
						val: "desc2",
					},
				},
				"1.3.6.1.2.1.2.2.1.20": {
					"1": snmpValue{
						val: float64(201),
					},
					"2": snmpValue{
						val: float64(202),
					},
				},
			},
			map[string]string{
				"1.3.6.1.2.1.2.2.1.14": "1.3.6.1.2.1.2.2.1.14.2",
				"1.3.6.1.2.1.2.2.1.2":  "1.3.6.1.2.1.2.2.1.2.2",
				"1.3.6.1.2.1.2.2.1.20": "1.3.6.1.2.1.2.2.1.20.2",
			},
		},
		{
			"no such object error case",
			[]string{"1.3.6.1.2.1.2.2.1.14", "1.3.6.1.2.1.2.2.1.2"},
			&gosnmp.SnmpPacket{
				Variables: []gosnmp.SnmpPDU{
					{
						Name: "1.3.6.1.2.1.2.2.1.14.1",
						Type: gosnmp.NoSuchObject,
					},
					{
						Name:  "1.3.6.1.2.1.2.2.1.2.1",
						Type:  gosnmp.OctetString,
						Value: []byte("desc1"),
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
				},
			},
			columnResultValuesType{
				"1.3.6.1.2.1.2.2.1.14": {
					// index 1 not fetched because of gosnmp.NoSuchObject error
					"2": snmpValue{
						val: float64(142),
					},
				},
				"1.3.6.1.2.1.2.2.1.2": {
					"1": snmpValue{
						val: "desc1",
					},
					"2": snmpValue{
						val: "desc2",
					},
				},
			},
			map[string]string{
				"1.3.6.1.2.1.2.2.1.14": "1.3.6.1.2.1.2.2.1.14.2",
				"1.3.6.1.2.1.2.2.1.2":  "1.3.6.1.2.1.2.2.1.2.2",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values, nextOidsMap := resultToColumnValues(tt.columnOids, tt.snmpPacket)
			assert.Equal(t, tt.expectedValues, values)
			assert.Equal(t, tt.expectedNextOidsMap, nextOidsMap)
		})
	}
}

func Test_resultToScalarValues(t *testing.T) {
	tests := []struct {
		name           string
		snmpPacket     *gosnmp.SnmpPacket
		expectedValues scalarResultValuesType
	}{
		{
			"simple case",
			&gosnmp.SnmpPacket{
				Variables: []gosnmp.SnmpPDU{
					{
						Name:  "1.3.6.1.2.1.2.2.1.14.1",
						Type:  gosnmp.Integer,
						Value: 142,
					},
					{
						Name:  "1.3.6.1.2.1.2.2.1.14.2",
						Type:  gosnmp.Counter32,
						Value: 142,
					},
					{
						Name:  "1.3.6.1.2.1.2.2.1.14.3",
						Type:  gosnmp.NoSuchInstance,
						Value: 142,
					},
				},
			},
			scalarResultValuesType{
				"1.3.6.1.2.1.2.2.1.14.1": {
					submissionType: metrics.GaugeType,
					val:            float64(142),
				},
				"1.3.6.1.2.1.2.2.1.14.2": {
					submissionType: metrics.RateType,
					val:            float64(142),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values := resultToScalarValues(tt.snmpPacket)
			assert.Equal(t, tt.expectedValues, values)
		})
	}
}
