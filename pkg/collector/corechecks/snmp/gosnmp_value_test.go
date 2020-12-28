package snmp

import (
	"github.com/DataDog/datadog-agent/pkg/util/log"
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
			snmpValue{valType: Other, val: float64(141)},
			nil,
		},
		{
			"OctetString",
			gosnmp.SnmpPDU{
				Name:  ".1.2.3",
				Type:  gosnmp.Integer,
				Value: 141,
			},
			"1.2.3",
			snmpValue{valType: Other, val: float64(141)},
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
			snmpValue{valType: Other, val: "myVal"},
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
			snmpValue{valType: Other, val: "myVal"},
			nil,
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
			log.Errorf("invalid type: Null"),
		},
		{
			"Counter32",
			gosnmp.SnmpPDU{
				Name:  ".1.2.3",
				Type:  gosnmp.Counter32,
				Value: uint(10),
			},
			"1.2.3",
			snmpValue{valType: Counter, val: float64(10)},
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
			snmpValue{valType: Other, val: float64(10)},
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
			snmpValue{valType: Other, val: float64(10)},
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
			snmpValue{valType: Counter, val: float64(10)},
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
			snmpValue{valType: Other, val: float64(10)},
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
			snmpValue{valType: Other, val: float64(10)},
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
			snmpValue{valType: Other, val: float64(10)},
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
			log.Errorf("invalid type: NoSuchObject"),
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
			log.Errorf("invalid type: NoSuchInstance"),
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
