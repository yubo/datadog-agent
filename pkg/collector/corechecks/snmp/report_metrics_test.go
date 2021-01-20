package snmp

import (
	"github.com/DataDog/datadog-agent/pkg/aggregator/mocksender"
	"github.com/DataDog/datadog-agent/pkg/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
)

func TestSendMetric(t *testing.T) {
	tests := []struct {
		caseName           string
		metricName         string
		value              snmpValueType
		tags               []string
		forcedType         string
		options            metricsConfigOption
		expectedMethod     string
		expectedMetricName string
		expectedValue      float64
		expectedTags       []string
		expectedSubMetrics int
	}{
		{
			caseName:           "Gauge metric case",
			metricName:         "gauge.metric",
			value:              snmpValueType{value: float64(10)},
			tags:               []string{},
			expectedMethod:     "Gauge",
			expectedMetricName: "snmp.gauge.metric",
			expectedValue:      float64(10),
			expectedTags:       []string{},
			expectedSubMetrics: 1,
		},
		{
			caseName:           "Counter32 metric case",
			metricName:         "counter.metric",
			value:              snmpValueType{submissionType: metrics.RateType, value: float64(10)},
			tags:               []string{},
			expectedMethod:     "Rate",
			expectedMetricName: "snmp.counter.metric",
			expectedValue:      float64(10),
			expectedTags:       []string{},
			expectedSubMetrics: 1,
		},
		{
			caseName:           "Forced gauge metric case",
			metricName:         "my.metric",
			value:              snmpValueType{submissionType: metrics.RateType, value: float64(10)},
			tags:               []string{},
			forcedType:         "gauge",
			expectedMethod:     "Gauge",
			expectedMetricName: "snmp.my.metric",
			expectedValue:      float64(10),
			expectedTags:       []string{},
			expectedSubMetrics: 1,
		},
		{
			caseName:           "Forced counter metric case",
			metricName:         "my.metric",
			value:              snmpValueType{submissionType: metrics.RateType, value: float64(10)},
			tags:               []string{},
			forcedType:         "counter",
			options:            metricsConfigOption{},
			expectedMethod:     "Rate",
			expectedMetricName: "snmp.my.metric",
			expectedValue:      float64(10),
			expectedTags:       []string{},
			expectedSubMetrics: 1,
		},
		{
			caseName:           "Forced monotonic_count metric case",
			metricName:         "my.metric",
			value:              snmpValueType{submissionType: metrics.RateType, value: float64(10)},
			tags:               []string{},
			forcedType:         "monotonic_count",
			options:            metricsConfigOption{},
			expectedMethod:     "MonotonicCount",
			expectedMetricName: "snmp.my.metric",
			expectedValue:      float64(10),
			expectedTags:       []string{},
			expectedSubMetrics: 1,
		},
		{
			caseName:           "Forced monotonic_count_and_rate metric case: MonotonicCount called",
			metricName:         "my.metric",
			value:              snmpValueType{submissionType: metrics.RateType, value: float64(10)},
			tags:               []string{},
			forcedType:         "monotonic_count_and_rate",
			options:            metricsConfigOption{},
			expectedMethod:     "MonotonicCount",
			expectedMetricName: "snmp.my.metric",
			expectedValue:      float64(10),
			expectedTags:       []string{},
			expectedSubMetrics: 2,
		},
		{
			caseName:           "Forced monotonic_count_and_rate metric case: Rate called",
			metricName:         "my.metric",
			value:              snmpValueType{submissionType: metrics.RateType, value: float64(10)},
			tags:               []string{},
			forcedType:         "monotonic_count_and_rate",
			options:            metricsConfigOption{},
			expectedMethod:     "Rate",
			expectedMetricName: "snmp.my.metric.rate",
			expectedValue:      float64(10),
			expectedTags:       []string{},
			expectedSubMetrics: 2,
		},
		{
			caseName:           "Forced percent metric case: Rate called",
			metricName:         "rate.metric",
			value:              snmpValueType{value: 0.5},
			tags:               []string{},
			forcedType:         "percent",
			options:            metricsConfigOption{},
			expectedMethod:     "Rate",
			expectedMetricName: "snmp.rate.metric",
			expectedValue:      50.0,
			expectedTags:       []string{},
			expectedSubMetrics: 1,
		},
		{
			caseName:           "Forced flag_stream case 1",
			metricName:         "metric",
			value:              snmpValueType{value: "1010"},
			tags:               []string{},
			forcedType:         "flag_stream",
			options:            metricsConfigOption{Placement: 1, MetricSuffix: "foo"},
			expectedMethod:     "Gauge",
			expectedMetricName: "snmp.metric.foo",
			expectedValue:      1.0,
			expectedTags:       []string{},
			expectedSubMetrics: 1,
		},
		{
			caseName:           "Forced flag_stream case 2",
			metricName:         "metric",
			value:              snmpValueType{value: "1010"},
			tags:               []string{},
			forcedType:         "flag_stream",
			options:            metricsConfigOption{Placement: 2, MetricSuffix: "foo"},
			expectedMethod:     "Gauge",
			expectedMetricName: "snmp.metric.foo",
			expectedValue:      0.0,
			expectedTags:       []string{},
			expectedSubMetrics: 1,
		},
		{
			caseName:           "Forced flag_stream invalid index",
			metricName:         "metric",
			value:              snmpValueType{value: "1010"},
			tags:               []string{},
			forcedType:         "flag_stream",
			options:            metricsConfigOption{Placement: 10, MetricSuffix: "foo"},
			expectedMethod:     "",
			expectedMetricName: "",
			expectedValue:      0.0,
			expectedTags:       []string{},
			expectedSubMetrics: 0,
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
			assert.Equal(t, tt.expectedSubMetrics, metricSender.submittedMetrics)
			if tt.expectedMethod != "" {
				mockSender.AssertCalled(t, tt.expectedMethod, tt.expectedMetricName, tt.expectedValue, "", tt.expectedTags)
			}
		})
	}
}
