package snmp

import (
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
	"testing"
)

func Test_transformIndex(t *testing.T) {
	tests := []struct {
		name               string
		indexes            []string
		transformRules     []metricIndexTransform
		expectedNewIndexes []string
	}{
		{
			"no rule",
			[]string{"10", "11", "12", "13"},
			[]metricIndexTransform{},
			nil,
		},
		{
			"one",
			[]string{"10", "11", "12", "13"},
			[]metricIndexTransform{
				{2, 3},
			},
			[]string{"12", "13"},
		},
		{
			"multi",
			[]string{"10", "11", "12", "13"},
			[]metricIndexTransform{
				{2, 2},
				{0, 1},
			},
			[]string{"12", "10", "11"},
		},
		{
			"out of index end",
			[]string{"10", "11", "12", "13"},
			[]metricIndexTransform{
				{2, 1000},
			},
			nil,
		},
		{
			"out of index start and end",
			[]string{"10", "11", "12", "13"},
			[]metricIndexTransform{
				{1000, 2000},
			},
			nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			newIndexes := transformIndex(tt.indexes, tt.transformRules)
			assert.Equal(t, tt.expectedNewIndexes, newIndexes)
		})
	}
}

func Test_metricsConfig_getTags(t *testing.T) {
	tests := []struct {
		name            string
		rawMetricConfig []byte
		fullIndex       string
		values          *snmpValues
		expectedTags    []string
	}{
		{
			"index transform",
			[]byte(`
table:
  OID:  1.2.3.4.5
  name: cpiPduBranchTable
symbols:
  - OID: 1.2.3.4.5.1.2
    name: cpiPduBranchCurrent
metric_tags:
  - column:
      OID:  1.2.3.4.8.1.2
      name: cpiPduName
    table: cpiPduTable
    index_transform:
      - start: 1
        end: 2
      - start: 6
        end: 7
    tag: pdu_name
`),
			"1.2.3.4.5.6.7.8",
			&snmpValues{
				columnValues: map[string]map[string]snmpValue{
					"1.2.3.4.8.1.2": {
						"2.3.7.8": snmpValue{
							val: "myval",
						},
					},
				},
			},
			[]string{"pdu_name:myval"},
		},
		{
			"index mapping",
			[]byte(`
table:
  OID: 1.3.6.1.2.1.4.31.3
  name: ipIfStatsTable
symbols:
  - OID: 1.3.6.1.2.1.4.31.3.1.6
    name: ipIfStatsHCInOctets
metric_tags:
  - index: 1
    tag: ipversion
    mapping:
      0: unknown
      1: ipv4
      2: ipv6
      3: ipv4z
      4: ipv6z
      16: dns
`),
			"3",
			&snmpValues{},
			[]string{"ipversion:ipv4z"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := metricsConfig{}
			yaml.Unmarshal(tt.rawMetricConfig, &m)

			tags := m.getTags(tt.fullIndex, tt.values)

			assert.Equal(t, tt.expectedTags, tags)
		})
	}
}
