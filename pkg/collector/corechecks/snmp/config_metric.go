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

	IndexTransform []metricIndexTransform `yaml:"index_transform"`

	Mapping map[string]string `yaml:"mapping"`
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

	// Legacy Symbol configs syntax
	OID  string `yaml:"OID"`
	Name string `yaml:"name"`

	// Table configs
	Table   symbolConfig   `yaml:"table"`
	Symbols []symbolConfig `yaml:"symbols"`

	MetricTags []metricTagConfig `yaml:"metric_tags"`

	ForcedType string              `yaml:"forced_type"`
	Options    metricsConfigOption `yaml:"options"`

	// TODO: Validate Symbol and Table are not both used
}

// getTags retrieve tags using the metric config and values
func (m *metricsConfig) getTags(fullIndex string, values *snmpValues) []string {
	var rowTags []string
	indexes := strings.Split(fullIndex, ".")
	for _, metricTag := range m.MetricTags {
		// get tag using `index` field
		if (metricTag.Index > 0) && (metricTag.Index <= uint(len(indexes))) {
			index := metricTag.Index - 1 // `index` metric config is 1-based
			if index >= uint(len(indexes)) {
				log.Debugf("error getting tags. index `%d` not found in indexes `%v`", metricTag.Index, indexes)
				continue
			}
			var tagValue string
			if len(metricTag.Mapping) > 0 {
				mappedValue, ok := metricTag.Mapping[indexes[index]]
				if !ok {
					log.Debugf("error getting tags. mapping for `%s` does not exist. mapping=`%v`, indexes=`%v`", indexes[index], metricTag.Mapping, indexes)
					continue
				}
				tagValue = mappedValue
			} else {
				tagValue = indexes[index]
			}
			rowTags = append(rowTags, metricTag.Tag+":"+tagValue)
		}
		// get tag using another column value
		if metricTag.Column.OID != "" {
			//tagValueOid := metricTag.Column.OID + "." + fullIndex
			stringValues, err := values.getColumnValues(metricTag.Column.OID)
			if err != nil {
				log.Debugf("error getting column value: %v", err)
				continue
			}

			// TODO: Test me (index transform code in getTags)
			var newIndexes []string
			if len(metricTag.IndexTransform) > 0 {
				newIndexes = transformIndex(indexes, metricTag.IndexTransform)
			} else {
				newIndexes = indexes
			}
			newFullIndex := strings.Join(newIndexes, ".")

			tagValue, ok := stringValues[newFullIndex]
			if !ok {
				// TODO: Test me
				log.Debugf("index not found for column value: tag=%v, index=%v", metricTag.Tag, newFullIndex)
			} else {
				rowTags = append(rowTags, metricTag.Tag+":"+tagValue.toString())
			}
		}
	}
	return rowTags
}

func transformIndex(indexes []string, transformRules []metricIndexTransform) []string {
	var newIndex []string

	for _, rule := range transformRules {
		start := rule.Start
		end := rule.End + 1
		if end > uint(len(indexes)) {
			return nil
		}
		newIndex = append(newIndex, indexes[start:end]...)
	}
	return newIndex
}

func normalizeMetrics(metrics []metricsConfig) {
	for i := range metrics {
		metric := &metrics[i]
		if metric.Symbol.Name == "" && metric.Symbol.OID == "" && metric.Name != "" && metric.OID != "" {
			metric.Symbol.Name = metric.Name
			metric.Symbol.OID = metric.OID
			metric.Name = ""
			metric.OID = ""
		}
	}
}
