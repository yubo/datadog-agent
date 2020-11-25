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
			sender := mocksender.NewMockSender("foo")
			sender.On("Gauge", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()
			sender.On("Rate", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()
			sendMetric(sender, tt.metricName, tt.value, tt.tags)
			sender.AssertCalled(t, tt.expectedMethod, tt.expectedMetricName, tt.expectedValue, "", tt.expectedTags)
		})
	}
}
