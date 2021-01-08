package snmp

import (
	"github.com/DataDog/datadog-agent/pkg/aggregator/mocksender"
	"github.com/DataDog/datadog-agent/pkg/metrics"
	"github.com/stretchr/testify/mock"
	"testing"
)

func TestSendMetric(t *testing.T) {
	tests := []struct {
		caseName           string
		metricName         string
		value              snmpValue
		tags               []string
		forcedType         string
		options            metricsConfigOption
		expectedMethod     string
		expectedMetricName string
		expectedValue      float64
		expectedTags       []string
	}{
		{
			"Gauge metric case",
			"gauge.metric",
			snmpValue{val: float64(10)},
			[]string{},
			"",
			metricsConfigOption{},
			"Gauge",
			"snmp.gauge.metric",
			float64(10),
			[]string{},
		},
		{
			"Counter32 metric case",
			"counter.metric",
			snmpValue{submissionType: metrics.RateType, val: float64(10)},
			[]string{},
			"",
			metricsConfigOption{},
			"Rate",
			"snmp.counter.metric",
			float64(10),
			[]string{},
		},
		{
			"Forced gauge metric case",
			"my.metric",
			snmpValue{submissionType: metrics.RateType, val: float64(10)},
			[]string{},
			"gauge",
			metricsConfigOption{},
			"Gauge",
			"snmp.my.metric",
			float64(10),
			[]string{},
		},
		{
			"Forced counter metric case",
			"my.metric",
			snmpValue{submissionType: metrics.RateType, val: float64(10)},
			[]string{},
			"counter",
			metricsConfigOption{},
			"Rate",
			"snmp.my.metric",
			float64(10),
			[]string{},
		},
		{
			"Forced monotonic_count metric case",
			"my.metric",
			snmpValue{submissionType: metrics.RateType, val: float64(10)},
			[]string{},
			"monotonic_count",
			metricsConfigOption{},
			"MonotonicCount",
			"snmp.my.metric",
			float64(10),
			[]string{},
		},
		{
			"Forced monotonic_count_and_rate metric case: MonotonicCount called",
			"my.metric",
			snmpValue{submissionType: metrics.RateType, val: float64(10)},
			[]string{},
			"monotonic_count_and_rate",
			metricsConfigOption{},
			"MonotonicCount",
			"snmp.my.metric",
			float64(10),
			[]string{},
		},
		{
			"Forced monotonic_count_and_rate metric case: Rate called",
			"my.metric",
			snmpValue{submissionType: metrics.RateType, val: float64(10)},
			[]string{},
			"monotonic_count_and_rate",
			metricsConfigOption{},
			"Rate",
			"snmp.my.metric.rate",
			float64(10),
			[]string{},
		},
		{
			"Forced percent metric case: Rate called",
			"rate.metric",
			snmpValue{val: 0.5},
			[]string{},
			"percent",
			metricsConfigOption{},
			"Rate",
			"snmp.rate.metric",
			50.0,
			[]string{},
		},
		{
			"Forced flag_stream case 1",
			"metric",
			snmpValue{val: "1010"},
			[]string{},
			"flag_stream",
			metricsConfigOption{Placement: 1, MetricSuffix: "foo"},
			"Gauge",
			"snmp.metric.foo",
			1.0,
			[]string{},
		},
		{
			"Forced flag_stream case 2",
			"metric",
			snmpValue{val: "1010"},
			[]string{},
			"flag_stream",
			metricsConfigOption{Placement: 2, MetricSuffix: "foo"},
			"Gauge",
			"snmp.metric.foo",
			0.0,
			[]string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.caseName, func(t *testing.T) {
			mockSender := mocksender.NewMockSender("foo")
			metricSender := metricSender{sender: mockSender}
			mockSender.On("MonotonicCount", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()
			mockSender.On("Gauge", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()
			mockSender.On("Rate", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()

			metricSender.sendMetric(tt.metricName, tt.value, tt.tags, tt.forcedType, tt.options)
			mockSender.AssertCalled(t, tt.expectedMethod, tt.expectedMetricName, tt.expectedValue, "", tt.expectedTags)
		})
	}
}
