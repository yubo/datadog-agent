package quantile

import (
	"github.com/DataDog/datadog-agent/pkg/trace/pb"
	"github.com/gogo/protobuf/proto"
)

// ddSketchToGK only support positive values
func ddSketchToGK(ddSketch *ddSketch) *SliceSummary {
	gkSketch := SliceSummary{Entries: make([]Entry, 0, ddSketch.nBuckets)}
	zeros := ddSketch.zeroCount
	if zeros > 0 {
		gkSketch.Entries = append(gkSketch.Entries, Entry{V: 0, G: zeros, Delta: 0})
	}
	total := zeros
	for ddSketch.iterator.hasNext() {
		index, g := ddSketch.iterator.next()
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
	// hitsDDSketch, err := newDDSketch([]*pb.DDSketch{&okSummary, &errorSummary})
	// if err != nil {
	// 	return nil, nil, err
	// }
	errorDDSketch, err := newDDSketch([]*pb.DDSketch{&errorSummary})
	if err != nil {
		return nil, nil, err
	}
	// return ddSketchToGK(hitsDDSketch), ddSketchToGK(errorDDSketch), nil
	return nil, ddSketchToGK(errorDDSketch), nil
}
