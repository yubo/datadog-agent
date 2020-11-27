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
			"Rate",
			"snmp.counter.metric",
			float64(10),
			[]string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.caseName, func(t *testing.T) {
			mockSender := mocksender.NewMockSender("foo")
			metricSender := metricSender{mockSender}
			mockSender.On("Gauge", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()
			mockSender.On("Rate", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()

			metricSender.sendMetric(tt.metricName, tt.value, tt.tags)
			mockSender.AssertCalled(t, tt.expectedMethod, tt.expectedMetricName, tt.expectedValue, "", tt.expectedTags)
		})
	}
}
