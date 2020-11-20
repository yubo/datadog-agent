package snmp

import "github.com/DataDog/datadog-agent/pkg/util/log"

/*
TODO: Shall we support 1/ deprecated syntax
We are only supporting case 2/ for now.

1/ deprecated
    metrics:
      - OID: 1.3.6.1.2.1.2.1
        name: ifNumber
2/
    metrics:
      - MIB: IF-MIB
        symbol:
          - OID: 1.3.6.1.2.1.2.1
            name: ifNumber
*/

type symbolConfig struct {
	OID  string `yaml:"OID"`
	Name string `yaml:"name"`
}

/*
metrics:
  # Example for the dummy table above:
  - MIB: EXAMPLE-MIB
    table:
      # Identification of the table which metrics come from.
      OID: 1.3.6.1.4.1.10
      name: exampleTable
    symbols:
      # List of symbols ('columns') to retrieve.
      # Same format as for a single OID.
      # Each row in the table will emit these metrics.
      - OID: 1.3.6.1.4.1.10.1.1
        name: exampleColumn1
      - OID: 1.3.6.1.4.1.10.1.2
        name: exampleColumn2
      # ...
*/

type metricTagConfig struct {
	Tag    string       `yaml:"tag"`
	Index  uint         `yaml:"index"`
	Column symbolConfig `yaml:"column"`
}

type metricsConfig struct {
	// Symbol configs
	Symbol symbolConfig `yaml:"symbol"`

	// Table configs
	Table   symbolConfig   `yaml:"table"`
	Symbols []symbolConfig `yaml:"symbols"`

	MetricTags []metricTagConfig `yaml:"metric_tags"`

	// TODO: Validate Symbol and Table are not both used
}

func (m *metricsConfig) getTags(indexes []string) []string {
	var rowTags []string
	for _, metricTag := range m.MetricTags {
		if (metricTag.Index == 0) || (metricTag.Index > uint(len(indexes))) {
			log.Warnf("invalid index %v, it must be between 1 and $v", metricTag.Index, len(indexes))
			continue
		}
		rowTags = append(rowTags, metricTag.Tag+":"+indexes[metricTag.Index-1])
	}
	return rowTags
}
