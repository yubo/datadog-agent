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

func (ms *metricSender) reportMetrics(metrics []metricsConfig, values *valueStoreType, tags []string) {
	for _, metric := range metrics {
		if metric.Symbol.OID != "" {
			ms.reportScalarMetrics(metric, values, tags)
		} else if metric.Table.OID != "" {
			ms.reportColumnMetrics(metric, values, tags)
		}
	}
}

func (ms *metricSender) getCheckInstanceMetricTags(metricTags []metricTagConfig, values *valueStoreType) []string {
	var globalTags []string

	for _, metricTag := range metricTags {
		value, err := values.getScalarValues(metricTag.OID)
		if err != nil {
			log.Warnf("error getting scalar value: %v", err)
			continue
		}
		globalTags = append(globalTags, metricTag.Tag+":"+value.toString())
	}
	return globalTags
}

func (ms *metricSender) reportScalarMetrics(metric metricsConfig, values *valueStoreType, tags []string) {
	value, err := values.getScalarValues(metric.Symbol.OID)
	if err != nil {
		log.Warnf("error getting scalar value: %v", err)
		return
	}
	ms.sendMetric(metric.Symbol.Name, value, tags, metric.ForcedType, metric.Options)
}

func (ms *metricSender) reportColumnMetrics(metricConfig metricsConfig, values *valueStoreType, tags []string) {
	for _, symbol := range metricConfig.Symbols {
		metricValues, err := values.getColumnValues(symbol.OID)
		if err != nil {
			log.Warnf("error getting column value: %v", err)
			continue
		}
		for fullIndex, value := range metricValues {
			rowTags := copyTags(tags)
			rowTags = append(rowTags, metricConfig.getTags(fullIndex, values)...)
			ms.sendMetric(symbol.Name, value, rowTags, metricConfig.ForcedType, metricConfig.Options)
			ms.trySendBandwidthUsageMetric(symbol, fullIndex, values, rowTags)
		}
	}
}

func (ms *metricSender) sendMetric(metricName string, value snmpValueType, tags []string, forcedType string, options metricsConfigOption) {
	metricFullName := "snmp." + metricName

	// we need copy tags before using sender due to https://github.com/DataDog/datadog-agent/issues/7159
	if forcedType != "" {
		switch forcedType {
		case "gauge":
			ms.gauge(metricFullName, value.toFloat64(), "", tags)
		case "counter":
			ms.rate(metricFullName, value.toFloat64(), "", tags)
		case "percent":
			ms.rate(metricFullName, value.toFloat64()*100, "", tags)
		case "monotonic_count":
			ms.monotonicCount(metricFullName, value.toFloat64(), "", tags)
		case "monotonic_count_and_rate":
			ms.monotonicCount(metricFullName, value.toFloat64(), "", tags)
			ms.rate(metricFullName+".rate", value.toFloat64(), "", tags)
		case "flag_stream":
			index := options.Placement - 1
			floatValue := 0.0
			if value.toString()[index] == '1' {
				floatValue = 1.0
			}
			ms.gauge(metricFullName+"."+options.MetricSuffix, floatValue, "", tags)
		default:
			// TODO: test me
			log.Warnf("metric `%s`: unsupported forcedType: %s", metricFullName, forcedType)
		}
	} else {
		switch value.submissionType {
		case metrics.RateType:
			ms.rate(metricFullName, value.toFloat64(), "", tags)
		default:
			ms.gauge(metricFullName, value.toFloat64(), "", tags)
		}
	}

	if forcedType == "monotonic_count_and_rate" {
		ms.submittedMetrics++
	}
	ms.submittedMetrics++
}

func (ms *metricSender) gauge(metric string, value float64, hostname string, tags []string) {
	// we need copy tags before using sender due to https://github.com/DataDog/datadog-agent/issues/7159
	ms.sender.Gauge(metric, value, hostname, copyTags(tags))
}

func (ms *metricSender) rate(metric string, value float64, hostname string, tags []string) {
	// we need copy tags before using sender due to https://github.com/DataDog/datadog-agent/issues/7159
	ms.sender.Rate(metric, value, hostname, copyTags(tags))
}

func (ms *metricSender) monotonicCount(metric string, value float64, hostname string, tags []string) {
	// we need copy tags before using sender due to https://github.com/DataDog/datadog-agent/issues/7159
	ms.sender.MonotonicCount(metric, value, hostname, copyTags(tags))
}

func (ms *metricSender) serviceCheck(checkName string, status metrics.ServiceCheckStatus, hostname string, tags []string, message string) {
	// we need copy tags before using sender due to https://github.com/DataDog/datadog-agent/issues/7159
	ms.sender.ServiceCheck(checkName, status, hostname, copyTags(tags), message)
}
