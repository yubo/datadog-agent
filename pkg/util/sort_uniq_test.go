// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2020 Datadog, Inc.

package util

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSortUniqInPlaceStringSlice(t *testing.T) {
	tags := NewStringSlice(3)
	tags.Append("tag2:tagval")
	tags.Append("tag3:value")
	tags.Append("zzz:tag")
	tags.Append("zzz:tag")
	tags.Append("a4:tagval")
	tags.Append("tag2:tagval")
	tags.Append("a1:tagval")
	SortUniqInPlace(tags)

	assert.ElementsMatch(t, tags.Slice(), []string{
		"a1:tagval",
		"a4:tagval",
		"tag2:tagval",
		"tag3:value",
		"zzz:tag",
	})
}

func benchmarkDeduplicateTags(b *testing.B, numberOfTags int) {
	// TODO(remy): restore
	// tags := make([]string, 0, numberOfTags+1)
	// for i := 0; i < numberOfTags; i++ {
	// 	tags = append(tags, fmt.Sprintf("aveeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeerylong:tag%d", i))
	// }
	// // this is the worst case for the insertion sort we are using
	// sort.Sort(sort.Reverse(sort.StringSlice(tags)))

	// tempTags := make([]string, len(tags))
	// copy(tempTags, tags)
	// b.ReportAllocs()
	// b.ResetTimer()

	// for n := 0; n < b.N; n++ {
	// 	copy(tempTags, tags)
	// 	SortUniqInPlace(tempTags)
	// }
}
func BenchmarkDeduplicateTags(b *testing.B) {
	for i := 1; i <= 128; i *= 2 {
		b.Run(fmt.Sprintf("deduplicate-%d-tags-in-place", i), func(b *testing.B) {
			benchmarkDeduplicateTags(b, i)
		})
	}
}
