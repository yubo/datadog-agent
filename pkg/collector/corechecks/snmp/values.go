package snmp

import (
	"fmt"
	"strconv"
)

type snmpValues struct {
	scalarValues map[string]interface{}
	columnValues map[string]map[string]interface{}
}

func toFloat64(value interface{}) float64 {
	var retValue float64

	switch value.(type) {
	case float64:
		retValue = value.(float64)
	case string:
		val, err := strconv.ParseInt(value.(string), 10, 64)
		if err != nil {
			return float64(0)
		}
		retValue = float64(val)
	}
	return retValue
}

// getScalarFloat64 look for oid and returns the value and boolean
// weather valid value has been found
func (v *snmpValues) getScalarFloat64(oid string) (float64, error) {
	value, ok := v.scalarValues[oid]
	if !ok {
		return float64(0), fmt.Errorf("value for Scalar OID not found: %s", oid)
	}
	return toFloat64(value), nil
}

func (v *snmpValues) getColumnValue(oid string) (map[string]float64, error) {
	retValues := make(map[string]float64)
	values, ok := v.columnValues[oid]
	if !ok {
		return nil, fmt.Errorf("value for Scalar OID not found: %s", oid)
	}
	for index, value := range values {
		retValues[index] = toFloat64(value)
	}

	return retValues, nil
}
