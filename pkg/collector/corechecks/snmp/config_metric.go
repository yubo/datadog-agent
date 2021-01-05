package snmp

import (
	"github.com/DataDog/datadog-agent/pkg/util/log"
	"strings"
)

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
	Tag string `yaml:"tag"`

	// Table config
	Index  uint         `yaml:"index"`
	Column symbolConfig `yaml:"column"`

	// Symbol config
	OID  string `yaml:"OID"`
	Name string `yaml:"symbol"`
}

type metricIndexTransform struct {
	Start uint `yaml:"start"`
	End   uint `yaml:"end"`
}

type metricsConfigOption struct {
	Placement    uint   `yaml:"placement"`
	MetricSuffix string `yaml:"metric_suffix"`
}

type metricsConfig struct {
	// Symbol configs
	Symbol symbolConfig `yaml:"symbol"`

	// Table configs
	Table   symbolConfig   `yaml:"table"`
	Symbols []symbolConfig `yaml:"symbols"`

	MetricTags []metricTagConfig `yaml:"metric_tags"`

	ForcedType string              `yaml:"forced_type"`
	Options    metricsConfigOption `yaml:"options"`

	IndexTransform []metricIndexTransform `yaml:"index_transform"`

	// TODO: Validate Symbol and Table are not both used
}

func (m *metricsConfig) getTags(fullIndex string, values *snmpValues) []string {
	var rowTags []string
	indexes := strings.Split(fullIndex, ".")
	for _, metricTag := range m.MetricTags {
		if (metricTag.Index > 0) && (metricTag.Index <= uint(len(indexes))) {
			rowTags = append(rowTags, metricTag.Tag+":"+indexes[metricTag.Index-1])
		}
		if metricTag.Column.OID != "" {
			//tagValueOid := metricTag.Column.OID + "." + fullIndex
			stringValues, err := values.getColumnValues(metricTag.Column.OID)
			if err != nil {
				log.Warnf("error getting column value: %v", err)
				continue
			}
			tagValue, ok := stringValues[fullIndex]
			if !ok {
				// TODO: Test me
				log.Debugf("index not found for column value: tag=%v, index=%v", metricTag.Tag, fullIndex)
			} else {
				rowTags = append(rowTags, metricTag.Tag+":"+tagValue.toString())
			}
		}
	}
	return rowTags
}
