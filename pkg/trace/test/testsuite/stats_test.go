package testsuite

import (
	"bytes"
	"fmt"
	"github.com/DataDog/sketches-go/ddsketch"
	"github.com/DataDog/sketches-go/ddsketch/mapping"
	"github.com/DataDog/sketches-go/ddsketch/store"
	"github.com/gogo/protobuf/proto"
	"github.com/tinylib/msgp/msgp"
	"math/rand"
	"net/http"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/DataDog/datadog-agent/pkg/trace/pb"
	"github.com/DataDog/datadog-agent/pkg/trace/test"
	"github.com/DataDog/datadog-agent/pkg/trace/test/testsuite/testdata"
)

func TestClientStats(t *testing.T) {
	var r test.Runner
	if err := r.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := r.Shutdown(time.Second); err != nil {
			t.Log("shutdown: ", err)
		}
	}()

	for _, tt := range testdata.ClientStatsTests {
		t.Run("", func(t *testing.T) {
			if err := r.RunAgent([]byte("hostname: agent-hostname\r\napm_config:\r\n  env: agent-env")); err != nil {
				t.Fatal(err)
			}
			defer r.KillAgent()

			if err := r.PostMsgpack("/v0.5/stats", &tt.In); err != nil {
				t.Fatal(err)
			}
			timeout := time.After(3 * time.Second)
			out := r.Out()
			for {
				select {
				case p := <-out:
					got, ok := p.(pb.StatsPayload)
					if !ok {
						continue
					}
					if reflect.DeepEqual(got, tt.Out) {
						return
					}
					t.Logf("got: %#v", got)
					t.Logf("expected: %#v", tt.Out)
					t.Fatal("did not match")
				case <-timeout:
					t.Fatalf("timed out, log was:\n%s", r.AgentLog())
				}
			}
		})
	}
}

func getEmptyDDSketch() []byte {
	m, _ := mapping.NewLogarithmicMapping(0.01)
	s := ddsketch.NewDDSketch(m, store.NewDenseStore(), store.NewDenseStore())
	data, _ := proto.Marshal(s.ToProto())
	return data
}

var testData = pb.ClientStatsPayload{
	Hostname: "testhost",
	Env:      "testing",
	Version:  "test-version",
	Stats: []pb.ClientStatsBucket{
		{
			Duration: uint64(time.Second.Nanoseconds()),
			Stats: []pb.ClientGroupedStats{
				{
					Service:        "test-hostname-service",
					Name:           "test-name",
					Resource:       "test-resource",
					HTTPStatusCode: 200,
					Type:           "web",
					Synthetics:     false,
					Hits:           1,
					Errors:         0,
					Duration:       10,
					OkSummary:      getEmptyDDSketch(),
					ErrorSummary:   getEmptyDDSketch(),
				},
			},
		},
	},
}

func TestHostnameAggregation(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	globalStart := time.Now().Truncate(time.Second*10).UnixNano()
	nPayloads := 100
	for i := 0; i < nPayloads; i++ {
		start := uint64(globalStart + rand.Int63n((time.Second*10).Nanoseconds()))
		testData.Stats[0].Start = start
		if err := postStats(&testData); err != nil {
			t.Fatal(err)
		}
	}
}

func postStats(data *pb.ClientStatsPayload) error {
	path := "/v0.5/stats"
	agentAddr := "localhost:8126"
	var buf bytes.Buffer
	if err := msgp.Encode(&buf, data); err != nil {
		return err
	}
	addr := fmt.Sprintf("http://%s%s", agentAddr, path)
	req, err := http.NewRequest("POST", addr, &buf)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/msgpack")
	req.Header.Set("Content-Length", strconv.Itoa(buf.Len()))
	req.Header.Set("Datadog-Meta-Tracer-Version", "0.2.0")
	req.Header.Set("Datadog-Meta-Lang", "go")
	client := &http.Client{
		Transport: &http.Transport{IdleConnTimeout: time.Second},
		Timeout:   time.Second,
	}
	_, err = client.Do(req)
	return err
}
