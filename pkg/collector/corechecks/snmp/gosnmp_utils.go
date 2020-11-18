package snmp

import (
	"github.com/soniah/gosnmp"
	"strings"
)

func getValueFromPDU(pduVariable gosnmp.SnmpPDU) (string, interface{}) {
	var value interface{}
	name := strings.TrimLeft(pduVariable.Name, ".") // remove leading dot
	switch pduVariable.Type {
	case gosnmp.OctetString:
		value = string(pduVariable.Value.([]byte))
	default:
		value = float64(gosnmp.ToBigInt(pduVariable.Value).Int64())
	}
	return name, value
}

func resultToScalarValues(result *gosnmp.SnmpPacket) (values map[string]interface{}) {
	// TODO: test me
	returnValues := make(map[string]interface{})
	for _, pduVariable := range result.Variables {
		name, value := getValueFromPDU(pduVariable)
		returnValues[name] = value
	}
	return returnValues
}

func resultToColumnValues(oids []string, result *gosnmp.SnmpPacket) (values map[string]map[string]interface{}) {
	// TODO: test me
	returnValues := make(map[string]map[string]interface{})
	for i, pduVariable := range result.Variables {
		name, value := getValueFromPDU(pduVariable)
		oid := oids[i%len(oids)]
		if _, ok := returnValues[oid]; !ok {
			returnValues[oids[i]] = make(map[string]interface{})
		}
		prefix := oid + "."
		if strings.HasPrefix(name, prefix) { // TODO: test me
			returnValues[oid][name[len(prefix):]] = value
		}
	}
	return returnValues
}
