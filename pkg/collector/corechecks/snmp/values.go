package snmp

import (
	"fmt"
	"strconv"
)

type valueType byte

// For now, we are only interested in Counter val type,
// this is needed a metric submission step to send metrics
// as `rate` submission type.
// Other is used as catch all, we will use `gauge` as submission type.
// Related Python integration code:
// https://github.com/DataDog/integrations-core/blob/51b1d2366b7cb7864c4b4aed29945ffd14e512d6/snmp/datadog_checks/snmp/metrics.py#L20-L21
const (
	Other valueType = iota
	Counter
)

type snmpValue struct {
	valType valueType
	val     interface{}
}

type snmpValues struct {
	scalarValues map[string]snmpValue
	columnValues map[string]map[string]snmpValue
}

func (sv *snmpValue) toFloat64() float64 {
	var retValue float64

	switch sv.val.(type) {
	case float64:
		retValue = sv.val.(float64)
	case string:
		val, err := strconv.ParseInt(sv.val.(string), 10, 64)
		if err != nil {
			return float64(0)
		}
		retValue = float64(val)
	}
	// TODO: only float64/string are expected. Probably no need to support other cases.
	return retValue
}

func (sv snmpValue) toString() string {
	var retValue string

	switch sv.val.(type) {
	case float64:
		retValue = strconv.Itoa(int(sv.val.(float64)))
	case string:
		retValue = sv.val.(string)
	}
	// TODO: only float64/string are expected. Probably no need to support other cases.
	return retValue
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
		return nil, fmt.Errorf("value for Scalar OID not found: %s", oid)
	}
	for index, value := range values {
		retValues[index] = value
	}

	return retValues, nil
}
