package tag

import (
	"sync/atomic"
)

var (
	// metaReady indicates whether the host metadata is available in the API
	metaReady int32
)

// MetaReady assigns the value v atomically.
func MetaReady(v int32) {
	atomic.StoreInt32(&metaReady, v)
}

// IsMetaReady returns true if the host metadata is available.
func IsMetaReady() bool {
	return (atomic.LoadInt32(&metaReady) == 1)
}
