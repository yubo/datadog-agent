package snmp

import (
	"github.com/soniah/gosnmp"
	"strings"
)

// resultToValues convert results into float and string value maps
func resultToValues(result *gosnmp.SnmpPacket) (floatValues map[string]float64, stringValues map[string]string) {
	floatValues = map[string]float64{}
	stringValues = map[string]string{}

	for _, pduVariable := range result.Variables {
		name := strings.TrimLeft(pduVariable.Name, ".") // remove leading dot
		switch pduVariable.Type {
		case gosnmp.OctetString:
			stringValues[name] = string(pduVariable.Value.([]byte))
		default:
			value := float64(gosnmp.ToBigInt(pduVariable.Value).Int64())
			floatValues[name] = value
		}
	}
	return floatValues, stringValues
}
