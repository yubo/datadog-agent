package api

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/DataDog/datadog-agent/pkg/trace/config"
	"github.com/DataDog/datadog-agent/pkg/trace/info"
	"github.com/DataDog/datadog-agent/pkg/trace/obfuscate"
)

// makeInfoHandler returns a new handler for handling the discovery endpoint.
func (r *HTTPReceiver) makeInfoHandler() (hash string, handler http.HandlerFunc) {
	var all []string
	for _, e := range endpoints {
		if !e.Hidden {
			all = append(all, e.Pattern)
		}
	}
	type reducedObfuscationConfig struct {
		ElasticSearch        bool                 `json:"elastic_search"`
		Mongo                bool                 `json:"mongo"`
		SQLExecPlan          bool                 `json:"sql_exec_plan"`
		SQLExecPlanNormalize bool                 `json:"sql_exec_plan_normalize"`
		HTTP                 obfuscate.HTTPConfig `json:"http"`
		RemoveStackTraces    bool                 `json:"remove_stack_traces"`
		Redis                bool                 `json:"redis"`
		Memcached            bool                 `json:"memcached"`
	}
	type reducedConfig struct {
		DefaultEnv             string                        `json:"default_env"`
		TargetTPS              float64                       `json:"target_tps"`
		MaxEPS                 float64                       `json:"max_eps"`
		ReceiverPort           int                           `json:"receiver_port"`
		ReceiverSocket         string                        `json:"receiver_socket"`
		ConnectionLimit        int                           `json:"connection_limit"`
		ReceiverTimeout        int                           `json:"receiver_timeout"`
		MaxRequestBytes        int64                         `json:"max_request_bytes"`
		StatsdPort             int                           `json:"statsd_port"`
		MaxMemory              float64                       `json:"max_memory"`
		MaxCPU                 float64                       `json:"max_cpu"`
		AnalyzedSpansByService map[string]map[string]float64 `json:"analyzed_spans_by_service"`
		Obfuscation            reducedObfuscationConfig      `json:"obfuscation"`
	}
	var oconf reducedObfuscationConfig
	if o := r.conf.Obfuscation; o != nil {
		oconf.ElasticSearch = o.ES.Enabled
		oconf.Mongo = o.Mongo.Enabled
		oconf.SQLExecPlan = o.SQLExecPlan.Enabled
		oconf.SQLExecPlanNormalize = o.SQLExecPlanNormalize.Enabled
		oconf.HTTP = o.HTTP
		oconf.RemoveStackTraces = o.RemoveStackTraces
		oconf.Redis = o.Redis.Enabled
		oconf.Memcached = o.Memcached.Enabled
	}
	txt, err := json.MarshalIndent(struct {
		Version       string        `json:"version"`
		GitCommit     string        `json:"git_commit"`
		BuildDate     string        `json:"build_date"`
		Endpoints     []string      `json:"endpoints"`
		FeatureFlags  []string      `json:"feature_flags,omitempty"`
		ClientDropP0s bool          `json:"client_drop_p0s"`
		Config        reducedConfig `json:"config"`
	}{
		Version:       info.Version,
		GitCommit:     info.GitCommit,
		BuildDate:     info.BuildDate,
		Endpoints:     all,
		FeatureFlags:  config.Features(),
		ClientDropP0s: true,
		Config: reducedConfig{
			DefaultEnv:             r.conf.DefaultEnv,
			TargetTPS:              r.conf.TargetTPS,
			MaxEPS:                 r.conf.MaxEPS,
			ReceiverPort:           r.conf.ReceiverPort,
			ReceiverSocket:         r.conf.ReceiverSocket,
			ConnectionLimit:        r.conf.ConnectionLimit,
			ReceiverTimeout:        r.conf.ReceiverTimeout,
			MaxRequestBytes:        r.conf.MaxRequestBytes,
			StatsdPort:             r.conf.StatsdPort,
			MaxMemory:              r.conf.MaxMemory,
			MaxCPU:                 r.conf.MaxCPU,
			AnalyzedSpansByService: r.conf.AnalyzedSpansByService,
			Obfuscation:            oconf,
		},
	}, "", "\t")
	if err != nil {
		panic(fmt.Errorf("Error making /info handler: %v", err))
	}
	h := sha256.Sum256(txt)
	return fmt.Sprintf("%x", h), func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, "%s", txt)
	}
}
