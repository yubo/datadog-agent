package quantile

import (
	"fmt"

	"github.com/DataDog/datadog-agent/pkg/trace/pb"
	"github.com/davecgh/go-spew/spew"
	"github.com/gogo/protobuf/proto"
)

// ddSketchToGK only support positive values
func ddSketchToGK(ddSketch *ddSketchReader) *SliceSummary {
	gkSketch := SliceSummary{Entries: make([]Entry, 0, ddSketch.nBuckets())}
	zeros := ddSketch.zeroCount
	if zeros > 0 {
		gkSketch.Entries = append(gkSketch.Entries, Entry{V: 0, G: zeros, Delta: 0})
	}
	total := zeros
	for ddSketch.hasNext() {
		index, g := ddSketch.next()
		if g == 0 {
			continue
		}
		total += g
		gkSketch.Entries = append(gkSketch.Entries, Entry{
			V:     ddSketch.mapping.Value(index),
			G:     g,
			Delta: int(2 * EPSILON * float64(total-1)),
		})
	}
	gkSketch.N = total
	if len(gkSketch.Entries) > 0 {
		gkSketch.Entries[0].Delta = 0
		gkSketch.Entries[len(gkSketch.Entries)-1].Delta = 0
	}
	gkSketch.compress()
	return &gkSketch
}

// DDSketchesToGK converts two dd sketches representing ok and errors to 2 gk sketches
// representing hits and errors, with hits = ok + errors
func DDSketchesToGK(okSummaryData []byte, errorSummaryData []byte) (hitsSketch *SliceSummary, errorSketch *SliceSummary, err error) {
	var okSummary pb.DDSketch
	if err := proto.Unmarshal(okSummaryData, &okSummary); err != nil {
		return nil, nil, err
	}
	var errorSummary pb.DDSketch
	if err := proto.Unmarshal(errorSummaryData, &errorSummary); err != nil {
		return nil, nil, err
	}
	okDDSketch, err := ddSketchReaderFromProto(&okSummary)
	if err != nil {
		return nil, nil, err
	}
	fmt.Println("\nok sketch")
	spew.Dump(okDDSketch)
	errorDDSketch, err := ddSketchReaderFromProto(&errorSummary)
	if err != nil {
		return nil, nil, err
	}
	fmt.Println("\nerror sketch")
	spew.Dump(errorDDSketch)
	hitsDDSketch := okDDSketch.merge(errorDDSketch)
	return ddSketchToGK(hitsDDSketch), ddSketchToGK(errorDDSketch), nil
}
