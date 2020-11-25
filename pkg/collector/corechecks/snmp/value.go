package snmp

import (
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
	// valType is used for knowing which default submission type
	// we should use (gauge, rate, etc)
	valType valueType
	// val might be a `string` or `float64`
	val interface{}
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
