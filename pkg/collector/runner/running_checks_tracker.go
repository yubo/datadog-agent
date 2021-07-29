// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package runner

import (
	"sync"

	"github.com/DataDog/datadog-agent/pkg/collector/check"
)

// withRunningChecksFunc is a closure you can run on a mutex-locked list of
// running checks
type withRunningChecksFunc func(map[check.ID]check.Check)

// RunningChecksTracker is an object that keeps a thread-safe track of
// all the running checks
type runningChecksTracker struct {
	runningChecks map[check.ID]check.Check // The list of checks running
	accessLock    sync.Mutex               // To control races on runningChecks
}

// NewRunningChecksTracker is a contructor for a RunningChecksTracker
func newRunningChecksTracker() *runningChecksTracker {
	return &runningChecksTracker{
		runningChecks: make(map[check.ID]check.Check),
	}
}

// Check returns a check in the running check list, if it can be found
func (t *runningChecksTracker) Check(id check.ID) (check.Check, bool) {
	t.accessLock.Lock()
	defer t.accessLock.Unlock()

	check, found := t.runningChecks[id]
	return check, found
}

// AddCheck adds a check to the list of running checks if the check
// isn't already added. Method returns a boolean if the addition was
// successful.
func (t *runningChecksTracker) AddCheck(check check.Check) bool {
	t.accessLock.Lock()
	defer t.accessLock.Unlock()

	if _, found := t.runningChecks[check.ID()]; found {
		return false
	}

	t.runningChecks[check.ID()] = check
	return true
}

// DeleteCheck removes a check from the list of running checks
func (t *runningChecksTracker) DeleteCheck(id check.ID) {
	t.accessLock.Lock()
	defer t.accessLock.Unlock()

	delete(t.runningChecks, id)
}

// WithRunningChecks takes in a function to execute in the context of a locked
// state of the checks tracker
func (t *runningChecksTracker) WithRunningChecks(closureFunc withRunningChecksFunc) {
	runningChecksCopy := t.RunningChecks()

	t.accessLock.Lock()
	defer t.accessLock.Unlock()

	closureFunc(runningChecksCopy)
}

// RunningChecks returns a list of all the running checks
func (t *runningChecksTracker) RunningChecks() map[check.ID]check.Check {
	t.accessLock.Lock()
	defer t.accessLock.Unlock()

	clone := make(map[check.ID]check.Check)
	for key, val := range t.runningChecks {
		clone[key] = val
	}

	return clone
}

// Lock acquires a lock on this object. useful in places where the values
// returned need to be processed without mutation.
func (t *runningChecksTracker) Lock() {
	t.accessLock.Lock()
}

// Unlock removes the lock holding this object
func (t *runningChecksTracker) Unlock() {
	t.accessLock.Unlock()
}
