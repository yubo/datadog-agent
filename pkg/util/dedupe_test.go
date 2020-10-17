package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDedupe(t *testing.T) {

	assert := assert.New(t)

	tags := []string{
		"zzz",
		"hello:world",
		"world:hello",
		"world:hello",
		"random2:value",
		"random1:value",
		"zzz",
	}

	deduped := Dedupe(tags)

	assert.NotEqual(len(tags), len(deduped))
	// two duplicates
	assert.Equal(len(tags)-2, len(deduped))

	m := make(map[string]int)
	for _, e := range tags {
		m[e] = 0
	}

	assert.Equal(len(m), len(deduped))

	for _, e := range deduped {
		if _, ok := m[e]; !ok {
			assert.Fail("Expected key to be found in map, but is unavailable")
		}
		m[e]++
	}

	for _, v := range m {
		assert.Equal(1, v)
	}

}
