package snmp

import (
	"github.com/DataDog/datadog-agent/pkg/metrics"
	"strconv"
)

type snmpValue struct {
	submissionType metrics.MetricType // used when sending the metric
	val            interface{}        // might be a `string` or `float64` type
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
