package snmp

import (
	"github.com/DataDog/datadog-agent/pkg/util/log"
	"github.com/soniah/gosnmp"
	"strings"
)

func getValueFromPDU(pduVariable gosnmp.SnmpPDU) (string, snmpValue, error) {
	var value interface{}
	name := strings.TrimLeft(pduVariable.Name, ".") // remove leading dot
	// TODO: Support:
	//   - gosnmp.Opaque ?
	//     Seems opaque type is never returned https://github.com/gosnmp/gosnmp/blob/dc320dac5b53d95a366733fd95fb5851f2099387/helper.go#L195-L205
	//   - gosnmp.Boolean: seems not exist anymore and not handled by gosnmp
	switch pduVariable.Type {
	case gosnmp.OctetString, gosnmp.BitString:
		value = string(pduVariable.Value.([]byte))
	case gosnmp.Integer, gosnmp.Counter32, gosnmp.Gauge32, gosnmp.TimeTicks, gosnmp.Counter64, gosnmp.Uinteger32:
		value = float64(gosnmp.ToBigInt(pduVariable.Value).Int64())
	case gosnmp.OpaqueFloat:
		value = float64(pduVariable.Value.(float32))
	case gosnmp.OpaqueDouble:
		value = pduVariable.Value.(float64)
	default:
		return name, snmpValue{}, log.Errorf("invalid type: %s", pduVariable.Type.String())
	}
	valueType := gosnmpTypeToSimpleType(pduVariable.Type)
	return name, snmpValue{valType: valueType, val: value}, nil
}

func resultToScalarValues(result *gosnmp.SnmpPacket) (values map[string]snmpValue) {
	// TODO: test me
	returnValues := make(map[string]snmpValue)
	for _, pduVariable := range result.Variables {
		name, value, err := getValueFromPDU(pduVariable)
		if err != nil {
			log.Debugf("Cannot get value for variable `%v` with type `%v` and value `%v`", pduVariable.Name, pduVariable.Type, pduVariable.Value)
			continue
		}
		returnValues[name] = value
	}
	return returnValues
}

func resultToColumnValues(columnOids []string, result *gosnmp.SnmpPacket) (map[string]map[string]snmpValue, map[string]string) {
	// TODO: test me
	returnValues := make(map[string]map[string]snmpValue)
	nextOidsMap := make(map[string]string)
	for i, pduVariable := range result.Variables {
		oid, value, err := getValueFromPDU(pduVariable)
		if err != nil {
			log.Debugf("Cannot get value for variable `%v` with type `%v` and value `%v`", pduVariable.Name, pduVariable.Type, pduVariable.Value)
			continue
		}
		columnOid := columnOids[i%len(columnOids)]
		if _, ok := returnValues[columnOid]; !ok {
			returnValues[columnOids[i]] = make(map[string]snmpValue)
		}
		prefix := columnOid + "."
		if strings.HasPrefix(oid, prefix) {
			returnValues[columnOid][oid[len(prefix):]] = value
			nextOidsMap[columnOid] = oid
		} else {
			// if oid is not prefixed by columnOid, it means it's not part of the column
			// and we can stop requesting the next row of this column
			delete(nextOidsMap, columnOid)
		}
	}
	return returnValues, nextOidsMap
}

func gosnmpTypeToSimpleType(gosnmpType gosnmp.Asn1BER) valueType {
	switch gosnmpType {
	case gosnmp.Counter32, gosnmp.Counter64:
		// TODO: ZeroBasedCounter64
		//   https://github.com/DataDog/integrations-core/blob/d6add1dfcd99c3610f45390b8d4cd97390af1f69/snmp/datadog_checks/snmp/pysnmp_inspect.py#L37-L38
		return Counter
	}
	return Other
}
