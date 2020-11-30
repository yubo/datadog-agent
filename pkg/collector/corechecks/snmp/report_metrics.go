package snmp

import (
	"github.com/DataDog/datadog-agent/pkg/aggregator"
	"github.com/DataDog/datadog-agent/pkg/util/log"
)

type metricSender struct {
	sender aggregator.Sender
}

func (ms *metricSender) reportMetrics(metrics []metricsConfig, metricTags []metricTagConfig, values *snmpValues, tags []string) {
	var newTags []string
	newTags = append(newTags, tags...)

	for _, metricTag := range metricTags {
		value, err := values.getScalarValues(metricTag.OID)
		if err != nil {
			log.Warnf("error getting scalar val: %v", err)
			continue
		}
		newTags = append(newTags, metricTag.Tag+":"+value.toString())
	}
	for _, metric := range metrics {
		if metric.Symbol.OID != "" {
			ms.reportScalarMetrics(metric, values, newTags)
		} else if metric.Table.OID != "" {
			ms.reportColumnMetrics(metric, values, newTags)
		}
	}
}

func (ms *metricSender) reportScalarMetrics(metric metricsConfig, values *snmpValues, tags []string) {
	value, err := values.getScalarValues(metric.Symbol.OID)
	if err != nil {
		log.Warnf("error getting scalar val: %v", err)
		return
	}
	ms.sendMetric(metric.Symbol.Name, value, tags)
}

func (ms *metricSender) reportColumnMetrics(metricConfig metricsConfig, values *snmpValues, tags []string) {
	for _, symbol := range metricConfig.Symbols {
		metricValues, err := values.getColumnValues(symbol.OID)
		if err != nil {
			log.Warnf("error getting column value: %v", err)
			continue
		}
		for fullIndex, value := range metricValues {
			var rowTags []string
			rowTags = append(rowTags, tags...)
			rowTags = append(rowTags, metricConfig.getTags(fullIndex, values)...)
			ms.sendMetric(symbol.Name, value, rowTags)
		}
		log.Infof("Table column %v - %v: %#v", symbol.Name, symbol.OID, values)
	}
}

func (ms *metricSender) sendMetric(metricName string, value snmpValue, tags []string) {
	// TODO: Submit using the right type
	//   See https://github.com/DataDog/integrations-core/blob/d6add1dfcd99c3610f45390b8d4cd97390af1f69/snmp/datadog_checks/snmp/pysnmp_inspect.py#L34-L48
	var senderFn func(metric string, value float64, hostname string, tags []string)
	switch value.valType {
	case Counter:
		senderFn = ms.sender.Rate
	default:
		senderFn = ms.sender.Gauge
	}
	senderFn("snmp."+metricName, value.toFloat64(), "", tags)
}
