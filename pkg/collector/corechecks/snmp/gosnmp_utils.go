package snmp

import (
	"github.com/soniah/gosnmp"
	"strings"
)

// resultToValues convert results into float and string value maps
func resultToValues(result *gosnmp.SnmpPacket) (values snmpValues) {
	returnValues := make(map[string]interface{})

	for _, pduVariable := range result.Variables {
		name := strings.TrimLeft(pduVariable.Name, ".") // remove leading dot
		switch pduVariable.Type {
		case gosnmp.OctetString:
			returnValues[name] = string(pduVariable.Value.([]byte))
		default:
			value := float64(gosnmp.ToBigInt(pduVariable.Value).Int64())
			returnValues[name] = value
		}
	}
	return snmpValues{returnValues}
}
