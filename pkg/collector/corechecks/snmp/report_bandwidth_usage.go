package snmp

import (
	"github.com/DataDog/datadog-agent/pkg/metrics"
	"github.com/DataDog/datadog-agent/pkg/util/log"
)

var bandwidthMetricNameToUsage = map[string]string{
	"ifHCInOctets":  "ifBandwidthInUsage",
	"ifHCOutOctets": "ifBandwidthOutUsage",
}

var ifHighSpeedOID = "1.3.6.1.2.1.31.1.1.1.15"

/*
   Evaluate and report input/output bandwidth usage. If any of `ifHCInOctets`, `ifHCOutOctets`  or `ifHighSpeed`
   is missing then bandwidth will not be reported.

   Bandwidth usage is:

   interface[In|Out]Octets(t+dt) - interface[In|Out]Octets(t)
   ----------------------------------------------------------
                   dt*interfaceSpeed

   Given:
   * ifHCInOctets: the total number of octets received on the interface.
   * ifHCOutOctets: The total number of octets transmitted out of the interface.
   * ifHighSpeed: An estimate of the interface's current bandwidth in Mb/s (10^6 bits
                  per second). It is constant in time, can be overwritten by the system admin.
                  It is the total available bandwidth.
   Bandwidth usage is evaluated as: ifHC[In|Out]Octets/ifHighSpeed and reported as *rate*
*/
func (ms *metricSender) sendBandwidthUsageMetric(symbol symbolConfig, fullIndex string, values *snmpValues, tags []string) {
	usageName, ok := bandwidthMetricNameToUsage[symbol.Name]
	if !ok {
		return
	}

	ifHighSpeedValues, err := values.getColumnValues(ifHighSpeedOID)
	if err != nil {
		log.Debugf("[SNMP Bandwidth usage] missing `ifHighSpeed` metric, skipping metric. fullIndex=%s", symbol.Name, fullIndex)
		return
	}

	metricValues, err := values.getColumnValues(symbol.OID)
	if err != nil {
		log.Debugf("[SNMP Bandwidth usage] missing `%s` metric, skipping this row. fullIndex=%s", symbol.Name, fullIndex)
		return
	}

	octetsValue, ok := metricValues[fullIndex]
	if !ok {
		log.Debugf("[SNMP Bandwidth usage] missing `%s` metric value, skipping this row. fullIndex=%s", symbol.Name, fullIndex)
		return
	}

	ifHighSpeedValue, ok := ifHighSpeedValues[fullIndex]
	if !ok {
		log.Debugf("[SNMP Bandwidth usage] missing `ifHighSpeed` metric value, skipping this row. fullIndex=%s", fullIndex)
		return
	}

	ifHighSpeedFloatValue := ifHighSpeedValue.toFloat64()
	if ifHighSpeedFloatValue == 0.0 {
		log.Debugf("[SNMP Bandwidth usage] Zero or invalid value for ifHighSpeed, skipping this row. fullIndex=%s, ifHighSpeedValue=%v", fullIndex, ifHighSpeedValue)
		return
	}
	usageValue := ((octetsValue.toFloat64() * 8) / (ifHighSpeedFloatValue * (1e6))) * 100.0

	ms.sendMetric(usageName+".rate", snmpValue{metrics.RateType, usageValue}, tags, "counter", metricsConfigOption{})
}
