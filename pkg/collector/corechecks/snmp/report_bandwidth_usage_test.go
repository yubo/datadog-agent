package snmp

import (
	"github.com/DataDog/datadog-agent/pkg/aggregator/mocksender"
	"github.com/DataDog/datadog-agent/pkg/metrics"
	"github.com/stretchr/testify/mock"
	"testing"
)

func Test_metricSender_sendBandwidthUsageMetric(t *testing.T) {
	type Metric struct {
		name  string
		value float64
	}
	tests := []struct {
		name           string
		symbol         symbolConfig
		fullIndex      string
		values         *snmpValues
		expectedMetric []Metric
	}{
		{
			"nominal",
			symbolConfig{OID: "1.3.6.1.2.1.31.1.1.1.6", Name: "ifHCInOctets"},
			"9",
			&snmpValues{
				columnValues: columnResultValuesType{
					// ifHCInOctets
					"1.3.6.1.2.1.31.1.1.1.6": map[string]snmpValue{
						"9": {
							metrics.GaugeType,
							5000000.0,
						},
					},
					// ifHCOutOctets
					"1.3.6.1.2.1.31.1.1.1.10": map[string]snmpValue{
						"9": {
							metrics.GaugeType,
							1000000.0,
						},
					},
					// ifHighSpeed
					"1.3.6.1.2.1.31.1.1.1.15": map[string]snmpValue{
						"9": {
							metrics.GaugeType,
							80.0,
						},
					},
				},
			},
			[]Metric{
				// ((5000000 * 8) / (80 * 1000000)) * 100 = 50.0
				{"snmp.ifBandwidthInUsage.rate", 50.0},
				//{"snmp.ifBandwidthOutUsage.rate", 10.0},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sender := mocksender.NewMockSender("testID") // required to initiate aggregator
			sender.On("Rate", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()

			ms := &metricSender{
				sender: sender,
			}
			tags := []string{"foo:bar"}
			ms.sendBandwidthUsageMetric(tt.symbol, tt.fullIndex, tt.values, tags)

			for _, metric := range tt.expectedMetric {
				sender.AssertMetric(t, "Rate", metric.name, metric.value, "", tags)
			}
		})
	}
}
