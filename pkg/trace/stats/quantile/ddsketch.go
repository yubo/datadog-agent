package quantile

import (
	"fmt"
	"github.com/DataDog/datadog-agent/pkg/trace/pb"
	"github.com/DataDog/sketches-go/ddsketch/mapping"
)

func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

func max(x, y int) int {
	if x > y {
		return x
	}
	return y
}

// ddSketchReader is used to iterator over the bins of a ddSketch.
// It allows merging of ddSketches without heavy allocations of bins.
// It only supports positive contiguous values.
type ddSketchReader struct {
	bins [][]float64
	maxIndexes []int
	minIndexes []int
	currentIndex int
	maxIndex int
	minIndex int
	zeroCount int
	mapping mapping.IndexMapping
}

func ddSketchReaderFromProto(sPb *pb.DDSketch) (s *ddSketchReader, err error) {
	s = &ddSketchReader{}
	s.bins = append(s.bins, sPb.PositiveValues.ContiguousBinCounts)
	s.minIndexes = append(s.minIndexes, int(sPb.PositiveValues.ContiguousBinIndexOffset))
	s.maxIndexes = append(s.maxIndexes, s.minIndexes[0] + len(s.bins[0]) - 1)
	s.maxIndex = s.maxIndexes[0]
	s.minIndex = s.minIndexes[0]
	s.currentIndex = s.minIndex
	s.zeroCount += int(sPb.ZeroCount)
	s.mapping, err = getDDSketchMapping(sPb.Mapping)
	return s, err
}

// merge merges r and r2 into a new ddSketchReader.
// it assumes that r and r2 are mergable (same gamma, same interpolation)
func (r *ddSketchReader) merge(r2 *ddSketchReader) *ddSketchReader {
	return &ddSketchReader{
		bins : append(r.bins, r2.bins...),
		maxIndexes : append(r.maxIndexes, r2.maxIndexes...),
		minIndexes : append(r.minIndexes, r2.minIndexes...),
		currentIndex : 0,
		maxIndex : max(r.maxIndex, r2.maxIndex),
		minIndex : min(r.minIndex, r2.minIndex),
		zeroCount : r.zeroCount + r2.zeroCount,
		mapping : r.mapping,
	}
}

func (r *ddSketchReader) nBuckets() int {
	return r.maxIndex - r.minIndex + 1
}

func (r *ddSketchReader) hasNext() bool {
	return r.currentIndex <= r.maxIndex
}

func (r *ddSketchReader) next() (index, count int) {
	index = r.currentIndex
	for j := 0; j < len(r.bins); j++ {
		if r.currentIndex >= r.minIndexes[j] && r.currentIndex <= r.maxIndexes[j] {
			count += int(r.bins[j][r.currentIndex-r.minIndexes[j]])
		}
	}
	r.currentIndex++
	return index, count
}

func getDDSketchMapping(protoMapping *pb.IndexMapping) (m mapping.IndexMapping, err error) {
	switch protoMapping.Interpolation {
	case pb.IndexMapping_NONE:
		return mapping.NewLogarithmicMappingWithGamma(protoMapping.Gamma, protoMapping.IndexOffset)
	case pb.IndexMapping_LINEAR:
		return mapping.NewLinearlyInterpolatedMappingWithGamma(protoMapping.Gamma, protoMapping.IndexOffset)
	case pb.IndexMapping_CUBIC:
		return mapping.NewCubicallyInterpolatedMappingWithGamma(protoMapping.Gamma, protoMapping.IndexOffset)
	default:
		return nil, fmt.Errorf("interpolation not supported: %d", protoMapping.Interpolation)
	}
}
