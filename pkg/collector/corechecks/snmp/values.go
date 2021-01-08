package snmp

import (
	"fmt"
)

type snmpValues struct {
	scalarValues scalarResultValuesType
	columnValues columnResultValuesType
}

// getScalarValues look for oid and returns the val and boolean
// weather valid value has been found
func (v *snmpValues) getScalarValues(oid string) (snmpValue, error) {
	value, ok := v.scalarValues[oid]
	if !ok {
		return snmpValue{}, fmt.Errorf("value for Scalar OID not found: %s", oid)
	}
	return value, nil
}

func (v *snmpValues) getColumnValues(oid string) (map[string]snmpValue, error) {
	retValues := make(map[string]snmpValue)
	values, ok := v.columnValues[oid]
	if !ok {
		return nil, fmt.Errorf("value for Column OID not found: %s", oid)
	}
	for index, value := range values {
		retValues[index] = value
	}

	return retValues, nil
}
