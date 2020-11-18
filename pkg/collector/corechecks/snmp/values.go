package snmp

import (
	"strconv"
)

type snmpValues struct {
	scalarValues map[string]interface{}
	columnValues map[string]map[string]interface{}
}

func newSnmpValues() *snmpValues {
	return &snmpValues{}
}

// getScalarFloat64 look for oid and returns the value and boolean
// weather valid value has been found
func (v *snmpValues) getScalarFloat64(oid string) (float64, bool) {
	value, ok := v.scalarValues[oid]
	if !ok {
		return float64(0), false
	}

	var retValue float64

	switch value.(type) {
	case float64:
		retValue = value.(float64)
	case string:
		val, err := strconv.ParseInt(value.(string), 10, 64)
		if err != nil {
			return float64(0), true
		}
		retValue = float64(val)
	}

	return retValue, true
}
