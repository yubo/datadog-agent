// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2020 Datadog, Inc.

package aggregator

import (
	"fmt"

	"github.com/DataDog/datadog-agent/pkg/aggregator/ckey"
	"github.com/DataDog/datadog-agent/pkg/metrics"
	"github.com/DataDog/datadog-agent/pkg/util"
	"github.com/DataDog/datadog-agent/pkg/util/log"
)

// Context holds the elements that form a context, and can be serialized into a context key
// Note that a Contenxt holding a StringSlice is responsible of pushing it back
// to the StringSlice pool.
type Context struct {
	Name string
	Tags *util.StringSlice
	Host string
}

// ContextResolver allows tracking and expiring contexts
type ContextResolver struct {
	contextsByKey map[ckey.ContextKey]*Context
	lastSeenByKey map[ckey.ContextKey]float64
	keyGenerator  *ckey.KeyGenerator
}

// generateContextKey generates the contextKey associated with the context of the metricSample
func (cr *ContextResolver) generateContextKey(metricSampleContext metrics.MetricSampleContext) ckey.ContextKey {
	return cr.keyGenerator.Generate(metricSampleContext.GetName(), metricSampleContext.GetHost(), metricSampleContext.GetTags())
}

func newContextResolver() *ContextResolver {
	return &ContextResolver{
		contextsByKey: make(map[ckey.ContextKey]*Context),
		lastSeenByKey: make(map[ckey.ContextKey]float64),
		keyGenerator:  ckey.NewKeyGenerator(),
	}
}

// trackContext returns the contextKey associated with the context of the metricSample and tracks that context
func (cr *ContextResolver) trackContext(metricSampleContext metrics.MetricSampleContext, currentTimestamp float64) (ckey.ContextKey, *Context) {
	// generate a key for this metric sample
	contextKey := cr.generateContextKey(metricSampleContext)
	// if this metric is not already tracked, create a context and start tracking it
	if _, ok := cr.contextsByKey[contextKey]; !ok {
		cr.contextsByKey[contextKey] = &Context{
			Name: metricSampleContext.GetName(),
			Tags: metricSampleContext.GetTags(),
			Host: metricSampleContext.GetHost(),
		}
	} else {
		// we can discard this StringSlice, it's not needed anymore because an
		// existing entry in the ContextResolver already contains these tags.
		util.GlobalStringSlicePool.Put(metricSampleContext.GetTags())
	}
	cr.lastSeenByKey[contextKey] = currentTimestamp
	return contextKey, cr.contextsByKey[contextKey]
}

// updateTrackedContext updates the last seen timestamp on a given context key
func (cr *ContextResolver) updateTrackedContext(contextKey ckey.ContextKey, timestamp float64) error {
	if _, ok := cr.lastSeenByKey[contextKey]; ok && cr.lastSeenByKey[contextKey] < timestamp {
		cr.lastSeenByKey[contextKey] = timestamp
	} else if !ok {
		return fmt.Errorf("Trying to update a context that is not tracked")
	}

	return nil
}

// expireContexts cleans up the contexts that haven't been tracked since the given timestamp
// and returns the associated contextKeys
func (cr *ContextResolver) expireContexts(expireTimestamp float64) []ckey.ContextKey {
	var expiredContextKeys []ckey.ContextKey

	// Find expired context keys
	for contextKey, lastSeen := range cr.lastSeenByKey {
		if lastSeen < expireTimestamp {
			expiredContextKeys = append(expiredContextKeys, contextKey)
		}
	}

	// Delete expired context keys
	for _, expiredContextKey := range expiredContextKeys {
		// the context is responsible of the StringSlice ownership
		ctx := cr.contextsByKey[expiredContextKey]
		util.GlobalStringSlicePool.Put(ctx.Tags)
		log.Info("Expiring", ctx, ctx.Tags.Slice())

		delete(cr.contextsByKey, expiredContextKey)
		delete(cr.lastSeenByKey, expiredContextKey)
	}

	return expiredContextKeys
}
