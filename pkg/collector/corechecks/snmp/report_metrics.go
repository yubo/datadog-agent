package snmp

import (
	"github.com/DataDog/datadog-agent/pkg/aggregator"
	"github.com/DataDog/datadog-agent/pkg/metrics"
	"github.com/DataDog/datadog-agent/pkg/util/log"
)

type metricSender struct {
	sender           aggregator.Sender
	submittedMetrics int
}

func (ms *metricSender) reportMetrics(metrics []metricsConfig, metricTags []metricTagConfig, values *snmpValues, tags []string) {
	// TODO: Move code to a better place, we should report `snmp.devices_monitored` even if calls fail
	ms.sender.Gauge("snmp.devices_monitored", float64(1), "", tags)
	for _, metric := range metrics {
		if metric.Symbol.OID != "" {
			ms.reportScalarMetrics(metric, values, tags)
		} else if metric.Table.OID != "" {
			ms.reportColumnMetrics(metric, values, tags)
		}
	}
}

func (ms *metricSender) getCheckInstanceMetricTags(metricTags []metricTagConfig, values *snmpValues) []string {
	var globalTags []string

	for _, metricTag := range metricTags {
		value, err := values.getScalarValues(metricTag.OID)
		if err != nil {
			log.Warnf("error getting scalar val: %v", err)
			continue
		}
		globalTags = append(globalTags, metricTag.Tag+":"+value.toString())
	}
	return globalTags
}

func (ms *metricSender) reportScalarMetrics(metric metricsConfig, values *snmpValues, tags []string) {
	value, err := values.getScalarValues(metric.Symbol.OID)
	if err != nil {
		log.Warnf("error getting scalar val: %v", err)
		return
	}
	ms.sendMetric(metric.Symbol.Name, value, tags, metric.ForcedType, metric.Options)
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
			ms.sendMetric(symbol.Name, value, rowTags, metricConfig.ForcedType, metricConfig.Options)
		}
	}
}

func (ms *metricSender) sendMetric(metricName string, value snmpValue, tags []string, forcedType string, options metricsConfigOption) {
	// TODO: Submit using the right type
	//   See https://github.com/DataDog/integrations-core/blob/d6add1dfcd99c3610f45390b8d4cd97390af1f69/snmp/datadog_checks/snmp/pysnmp_inspect.py#L34-L48
	metricFullName := "snmp." + metricName

	if forcedType != "" {
		switch forcedType {
		case "gauge":
			ms.sender.Gauge(metricFullName, value.toFloat64(), "", tags)
		case "counter":
			ms.sender.Rate(metricFullName, value.toFloat64(), "", tags)
		case "percent":
			ms.sender.Rate(metricFullName, value.toFloat64()*100, "", tags)
		case "monotonic_count":
			ms.sender.MonotonicCount(metricFullName, value.toFloat64(), "", tags)
		case "monotonic_count_and_rate":
			ms.sender.MonotonicCount(metricFullName, value.toFloat64(), "", tags)
			ms.sender.Rate(metricFullName+".rate", value.toFloat64(), "", tags)
		case "flag_stream":
			index := options.Placement - 1
			floatValue := 0.0
			if value.toString()[index] == '1' {
				floatValue = 1.0
			}
			ms.sender.Gauge(metricFullName+"."+options.MetricSuffix, floatValue, "", tags)
		default:
			// TODO: test me
			log.Warnf("metric `%s`: unsupported forcedType: %s", metricFullName, forcedType)
		}
	} else {
		switch value.submissionType {
		case metrics.RateType:
			ms.sender.Rate(metricFullName, value.toFloat64(), "", tags)
		default:
			ms.sender.Gauge(metricFullName, value.toFloat64(), "", tags)
		}
	}

	ms.submittedMetrics++
}
