package snmp

import (
	"fmt"
	"github.com/DataDog/datadog-agent/pkg/util/log"
	"github.com/soniah/gosnmp"
	"strings"
)

// getValueFromPDU converts gosnmp.SnmpPDU to snmpValue
// See possible types here: https://github.com/gosnmp/gosnmp/blob/master/helper.go#L59-L271
func getValueFromPDU(pduVariable gosnmp.SnmpPDU) (string, snmpValue, error) {
	var value interface{}
	name := strings.TrimLeft(pduVariable.Name, ".") // remove leading dot
	// TODO: Support:
	//   - gosnmp.Opaque ?
	//     Seems opaque type is never returned https://github.com/gosnmp/gosnmp/blob/dc320dac5b53d95a366733fd95fb5851f2099387/helper.go#L195-L205
	//   - gosnmp.Boolean: seems not exist anymore and not handled by gosnmp
	//   - gosnmp.EndOfMibView

	// TODO: Handle error if pduVariable.Value has a wrong type for each switch case
	switch pduVariable.Type {
	case gosnmp.OctetString, gosnmp.BitString:
		// TODO: Might need better impl.
		//   We hexify like Python/pysnmp impl (keep compatibility) if the value contains non ascii letters:
		//   https://github.com/etingof/pyasn1/blob/db8f1a7930c6b5826357646746337dafc983f953/pyasn1/type/univ.py#L950-L953
		//   hexifying like pysnmp prettyPrint might lead to unpredictable results since `[]byte` might or might not have
		//   elements outside of 32-126 range
		//   An alternative solution is to explicitly force the conversion to specific type using profile config.
		bytesValue := pduVariable.Value.([]byte)
		if hasNonPrintableByte(bytesValue) {
			value = fmt.Sprintf("%#x", bytesValue)
		} else {
			value = string(bytesValue)
		}
	case gosnmp.Integer, gosnmp.Counter32, gosnmp.Gauge32, gosnmp.TimeTicks, gosnmp.Counter64, gosnmp.Uinteger32:
		value = float64(gosnmp.ToBigInt(pduVariable.Value).Int64())
	case gosnmp.OpaqueFloat:
		value = float64(pduVariable.Value.(float32))
	case gosnmp.OpaqueDouble:
		value = pduVariable.Value.(float64)
	case gosnmp.IPAddress:
		strValue, ok := pduVariable.Value.(string)
		if !ok {
			return name, snmpValue{}, fmt.Errorf("oid %s: invalid IP Address with value %v", pduVariable.Name, pduVariable.Value)
		}
		value = strValue
	case gosnmp.ObjectIdentifier:
		value = strings.TrimLeft(pduVariable.Value.(string), ".")
	default:
		return name, snmpValue{}, fmt.Errorf("oid %s: invalid type: %s", pduVariable.Name, pduVariable.Type.String())
	}
	valueType := gosnmpTypeToSimpleType(pduVariable.Type)
	return name, snmpValue{valType: valueType, val: value}, nil
}

func hasNonPrintableByte(bytesValue []byte) bool {
	hasNonPrintable := false

	for _, bit := range bytesValue {
		if bit < 32 || bit > 126 {
			hasNonPrintable = true
		}
	}
	return hasNonPrintable
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

func resultToColumnValues(columnOids []string, snmpPacket *gosnmp.SnmpPacket) (map[string]map[string]snmpValue, map[string]string) {
	returnValues := make(map[string]map[string]snmpValue)
	nextOidsMap := make(map[string]string)
	for i, pduVariable := range snmpPacket.Variables {
		oid, value, err := getValueFromPDU(pduVariable)
		if err != nil {
			log.Debugf("Cannot get value for variable `%v` with type `%v` and value `%v`", pduVariable.Name, pduVariable.Type, pduVariable.Value)
			continue
		}
		columnOid := columnOids[i%len(columnOids)]
		if _, ok := returnValues[columnOid]; !ok {
			returnValues[columnOid] = make(map[string]snmpValue)
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

// gosnmpTypeToSimpleType converts gosnmp.Asn1BER type to valueType.
// The simple type is used to know what type to use when submitting metrics.
func gosnmpTypeToSimpleType(gosnmpType gosnmp.Asn1BER) valueType {
	switch gosnmpType {
	case gosnmp.Counter32, gosnmp.Counter64:
		// TODO: ZeroBasedCounter64
		//   https://github.com/DataDog/integrations-core/blob/d6add1dfcd99c3610f45390b8d4cd97390af1f69/snmp/datadog_checks/snmp/pysnmp_inspect.py#L37-L38
		return Counter
	}
	return Other
}