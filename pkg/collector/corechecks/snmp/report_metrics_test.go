package snmp

import (
	"github.com/DataDog/datadog-agent/pkg/aggregator/mocksender"
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
		expectedMethod     string
		expectedMetricName string
		expectedValue      float64
		expectedTags       []string
	}{
		{
			"Gauge metric case",
			"gauge.metric",
			snmpValue{valType: Other, val: float64(10)},
			[]string{},
			"",
			"Gauge",
			"snmp.gauge.metric",
			float64(10),
			[]string{},
		},
		{
			"Counter32 metric case",
			"counter.metric",
			snmpValue{valType: Counter, val: float64(10)},
			[]string{},
			"",
			"Rate",
			"snmp.counter.metric",
			float64(10),
			[]string{},
		},
		{
			"Forced gauge metric case",
			"my.metric",
			snmpValue{valType: Counter, val: float64(10)},
			[]string{},
			"gauge",
			"Gauge",
			"snmp.my.metric",
			float64(10),
			[]string{},
		},
		{
			"Forced counter metric case",
			"my.metric",
			snmpValue{valType: Counter, val: float64(10)},
			[]string{},
			"counter",
			"Rate",
			"snmp.my.metric",
			float64(10),
			[]string{},
		},
		{
			"Forced monotonic_count metric case",
			"my.metric",
			snmpValue{valType: Counter, val: float64(10)},
			[]string{},
			"monotonic_count",
			"MonotonicCount",
			"snmp.my.metric",
			float64(10),
			[]string{},
		},
		{
			"Forced monotonic_count_and_rate metric case: MonotonicCount called",
			"my.metric",
			snmpValue{valType: Counter, val: float64(10)},
			[]string{},
			"monotonic_count_and_rate",
			"MonotonicCount",
			"snmp.my.metric",
			float64(10),
			[]string{},
		},
		{
			"Forced monotonic_count_and_rate metric case: Rate called",
			"my.metric",
			snmpValue{valType: Counter, val: float64(10)},
			[]string{},
			"monotonic_count_and_rate",
			"Rate",
			"snmp.my.metric.rate",
			float64(10),
			[]string{},
		},
		{
			"Forced percent metric case: Rate called",
			"rate.metric",
			snmpValue{valType: Other, val: 0.5},
			[]string{},
			"percent",
			"Rate",
			"snmp.rate.metric",
			50.0,
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

			metricSender.sendMetric(tt.metricName, tt.value, tt.tags, tt.forcedType)
			mockSender.AssertCalled(t, tt.expectedMethod, tt.expectedMetricName, tt.expectedValue, "", tt.expectedTags)
		})
	}
}
