package snmp

import "github.com/DataDog/datadog-agent/pkg/aggregator"

func sendMetric(sender aggregator.Sender, metricName string, value snmpValue, tags []string) {
	// TODO: Submit using the right type
	//   See https://github.com/DataDog/integrations-core/blob/d6add1dfcd99c3610f45390b8d4cd97390af1f69/snmp/datadog_checks/snmp/pysnmp_inspect.py#L34-L48
	var senderFn func(metric string, value float64, hostname string, tags []string)
	switch value.valType {
	case Counter:
		senderFn = sender.Rate
	default:
		senderFn = sender.Gauge
	}
	senderFn("snmp."+metricName, value.toFloat64(), "", tags)
}
