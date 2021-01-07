// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2020 Datadog, Inc.

package metrics

import (
	"github.com/DataDog/datadog-agent/pkg/util/log"
)

// Gauge tracks the value of a metric
type Gauge struct {
	gauge   float64
	sampled bool
}

func (g *Gauge) addSample(sample *MetricSample, timestamp float64) {
	log.Infof("Gauge add: sample.Name: %v, sample.Value: %v, sample.RawValue: %v", sample.Name, sample.Value, sample.RawValue)
	g.gauge = sample.Value
	g.sampled = true
}

func (g *Gauge) flush(timestamp float64) ([]*Serie, error) {
	value, sampled := g.gauge, g.sampled
	log.Infof("Gauge flush: value: %v, sampled: %v", value, sampled)
	g.gauge, g.sampled = 0, false

	if !sampled {
		return []*Serie{}, NoSerieError{}
	}

	return []*Serie{
		{
			// we use the timestamp passed to the flush
			Points: []Point{{Ts: timestamp, Value: value}},
			MType:  APIGaugeType,
		},
	}, nil
}
