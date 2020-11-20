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
		// TODO: Why gosnmp examples are using `string(pduVariable.Value.([]byte))`
		//   but `pduVariable.Value.(string)` is the one that works
		value = pduVariable.Value.(string)
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

func resultToColumnValues(oids []string, result *gosnmp.SnmpPacket) (map[string]map[string]interface{}, map[string]string) {
	// TODO: test me
	returnValues := make(map[string]map[string]interface{})
	nextOidsMap := make(map[string]string)
	for i, pduVariable := range result.Variables {
		name, value := getValueFromPDU(pduVariable)
		oid := oids[i%len(oids)]
		if _, ok := returnValues[oid]; !ok {
			returnValues[oids[i]] = make(map[string]interface{})
		}
		prefix := oid + "."
		if strings.HasPrefix(name, prefix) {
			returnValues[oid][name[len(prefix):]] = value
			nextOidsMap[oid] = name
		} else {
			delete(nextOidsMap, oid)
		}
	}
	return returnValues, nextOidsMap
}
