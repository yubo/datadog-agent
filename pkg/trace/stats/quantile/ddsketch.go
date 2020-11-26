package quantile

import (
	"fmt"
	"github.com/DataDog/datadog-agent/pkg/trace/pb"
	"github.com/DataDog/sketches-go/ddsketch/mapping"
)

// ddSketchIterator is an iterator on a list of ddSketches.
// it's a way to "merge" sketches without allocating any memory
// it only supports positive contiguousBins
type ddSketchIterator struct {
	bins [][]float64
	maxIndexes []int
	minIndexes []int
	currentIndex int
	maxIndex int
	minIndex int
}

func (i *ddSketchIterator) nBuckets() int {
	return i.maxIndex - i.minIndex + 1
}

func (i *ddSketchIterator) hasNext() bool {
	return i.currentIndex <= i.maxIndex
}

func (i *ddSketchIterator) next() (index, count int) {
	index = i.currentIndex
	for j := 0; j < len(i.bins); j++ {
		if i.currentIndex >= i.minIndexes[j] && i.currentIndex <= i.maxIndexes[j] {
			count += int(i.bins[j][i.currentIndex-i.minIndexes[j]])
		}
	}
	i.currentIndex++
	return index, count
}

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

func newDDSketchIterator(bins []float64, offset int) ddSketchIterator {
	i := ddSketchIterator{}
	i.bins = append(i.bins, bins)
	i.minIndexes = append(i.minIndexes, offset)
	i.maxIndexes = append(i.maxIndexes, offset + len(bins) - 1)
	i.maxIndex = i.maxIndexes[0]
	i.minIndex = i.minIndexes[0]
	i.currentIndex = i.minIndex
	return i
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

type ddSketchReader struct {
	ddSketchIterator
	zeroCount int
	mapping mapping.IndexMapping
}

// merge merges r and r2 into a new ddSketchReader.
// It's not reallocating the bins
// it assumes that r and r2 are mergable (same gamma, same interpolation)
func (r *ddSketchReader) merge(r2 *ddSketchReader) *ddSketchReader {
	merged := ddSketchReader{}
	merged.bins = append(r.bins, r2.bins...)
	merged.maxIndexes = append(r.maxIndexes, r2.maxIndexes...)
	merged.minIndexes = append(r.minIndexes, r2.minIndexes...)
	merged.currentIndex = 0
	merged.maxIndex = max(r.maxIndex, r2.maxIndex)
	merged.minIndex = min(r.minIndex, r2.minIndex)
	merged.zeroCount = r.zeroCount + r2.zeroCount
	merged.mapping = r.mapping
	return &merged
}

func ddSketchReaderFromProto(s *pb.DDSketch) (sketch *ddSketchReader, err error) {
	sketch = &ddSketchReader{
		ddSketchIterator: newDDSketchIterator(s.PositiveValues.ContiguousBinCounts, int(s.PositiveValues.ContiguousBinIndexOffset)),
	}
	sketch.zeroCount += int(s.ZeroCount)
	sketch.mapping, err = getDDSketchMapping(s.Mapping)
	return sketch, err
}
