// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2020 Datadog, Inc.

package metrics

import (
	"github.com/DataDog/datadog-agent/pkg/util/log"
)

// Count is used to count the number of events that occur between 2 flushes. Each sample's value is added
// to the value that's flushed
type Count struct {
	value   float64
	sampled bool
}

func (c *Count) addSample(sample *MetricSample, timestamp float64) {
	log.Infof("Count add: sample.Name: %v, sample.Value: %v, sample.RawValue: %v", sample.Name, sample.Value, sample.RawValue)
	c.value += sample.Value
	c.sampled = true
}

func (c *Count) flush(timestamp float64) ([]*Serie, error) {
	value, sampled := c.value, c.sampled
	log.Infof("Count flush: value: %v, sampled: %v", value, sampled)
	c.value, c.sampled = 0, false

	if !sampled {
		return []*Serie{}, NoSerieError{}
	}

	return []*Serie{
		{
			// we use the timestamp passed to the flush
			Points: []Point{{Ts: timestamp, Value: value}},
			MType:  APICountType,
		},
	}, nil
}
