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

func min(values []int) int {
	min := values[0]
	for i := 1; i < len(values); i++ {
		if values[i] < min {
			min = values[i]
		}
	}
	return min
}

func max(values []int) int {
	max := values[0]
	for i := 1; i < len(values); i++ {
		if values[i] > max {
			max = values[i]
		}
	}
	return max
}

func newDDSketchIterator(sketches []*pb.DDSketch) ddSketchIterator {
	i := ddSketchIterator{
		bins: make([][]float64, len(sketches)),
		minIndexes: make([]int, len(sketches)),
		maxIndexes: make([]int, len(sketches)),
	}
	for j := 0; j < len(sketches); j++ {
		i.bins[j] = sketches[j].PositiveValues.ContiguousBinCounts
		i.minIndexes[j] = int(sketches[j].PositiveValues.ContiguousBinIndexOffset)
		i.maxIndexes[j] = i.minIndexes[j] + len(i.bins[j]) - 1
	}
	i.maxIndex = max(i.maxIndexes)
	i.minIndex = min(i.minIndexes)
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
	// nBuckets is the number of buckets, including empty buckets
	nBuckets int
	mapping mapping.IndexMapping
}

func newDDSketch(sketches []*pb.DDSketch) (sketch *ddSketchReader, err error) {
	sketch = &ddSketchReader{
		ddSketchIterator: newDDSketchIterator(sketches),
	}
	for _, s := range sketches {
		sketch.zeroCount += int(s.ZeroCount)
	}
	sketch.nBuckets = sketch.maxIndex - sketch.minIndex + 1
	sketch.mapping, err = getDDSketchMapping(sketches[0].Mapping)
	return sketch, err
}
