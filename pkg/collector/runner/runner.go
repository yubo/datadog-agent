// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package runner

import (
	"fmt"

	"sync"
	"sync/atomic"
	"time"

	"github.com/DataDog/datadog-agent/pkg/collector/check"
	"github.com/DataDog/datadog-agent/pkg/collector/scheduler"
	"github.com/DataDog/datadog-agent/pkg/config"
	"github.com/DataDog/datadog-agent/pkg/util/log"
)

const (
	// Time to wait for a check to stop
	stopCheckTimeout time.Duration = 500 * time.Millisecond
	// Time to wait for all checks to stop
	stopAllChecksTimeout time.Duration = 2 * time.Second
)

var (
	// TestWg is used for testing the number of check workers
	TestWg sync.WaitGroup

	// Atomic incrementing variables for generating globally unique runner and worker object IDs
	runnerIDGenerator uint64
	workerIDGenerator uint64
)

// Runner is the object in charge of running all the checks
type Runner struct {
	// keep members that are used in atomic functions at the top of the structure
	// important for 32 bit compiles.
	// see https://github.com/golang/go/issues/599#issuecomment-419909701 for more information
	isRunning uint32 // Flag to see if the Runner is, well, running

	id                  int                   // Globally unique identifier for the Runner
	workers             map[int]*Worker       // Workers currrently under this Runner's management
	workersLock         sync.Mutex            // Lock to prevent concurrent worker changes
	isStaticWorkerCount bool                  // Flag indicating if numWorkers is dynamically updated
	pendingCheckChan    chan check.Check      // The channel where checks come from
	checksTracker       *runningChecksTracker // Tracker in charge of maintaining the running check list
	scheduler           *scheduler.Scheduler  // Scheduler runner operates on
	schedulerLock       sync.Mutex            // Lock around operations on the scheduler
}

// NewRunner takes the number of desired goroutines processing incoming checks.
func NewRunner() *Runner {
	numWorkers := config.Datadog.GetInt("check_runners")

	r := &Runner{
		id:                  int(atomic.AddUint64(&runnerIDGenerator, 1)),
		isRunning:           1,
		workers:             make(map[int]*Worker),
		isStaticWorkerCount: numWorkers != 0,
		pendingCheckChan:    make(chan check.Check),
		checksTracker:       newRunningChecksTracker(),
	}

	if !r.isStaticWorkerCount {
		numWorkers = config.DefaultNumWorkers
	}

	r.ensureMinWorkers(numWorkers)

	return r
}

// EnsureMinWorkers increases the number of workers to match the
// `desiredNumWorkers` parameter
func (r *Runner) ensureMinWorkers(desiredNumWorkers int) {
	r.workersLock.Lock()
	defer r.workersLock.Unlock()

	currentWorkers := len(r.workers)

	if desiredNumWorkers <= currentWorkers {
		return
	}

	workersToAdd := desiredNumWorkers - currentWorkers
	for idx := 0; idx < workersToAdd; idx++ {
		worker := r.newWorker()
		r.workers[worker.ID] = worker
	}

	log.Infof(
		"Runner %d added %d workers (total: %d)",
		r.id,
		workersToAdd,
		len(r.workers),
	)
}

// AddWorker adds a single worker to the runner
func (r *Runner) AddWorker() {
	r.workersLock.Lock()
	defer r.workersLock.Unlock()

	worker := r.newWorker()
	r.workers[worker.ID] = worker
}

// AddWorker adds a new worker running in a separate goroutine
func (r *Runner) newWorker() *Worker {
	worker := &Worker{
		ID:                     int(atomic.AddUint64(&workerIDGenerator, 1)),
		ChecksTracker:          r.checksTracker,
		PendingCheckChan:       r.pendingCheckChan,
		RunnerID:               r.id,
		ShouldAddWorkStatsFunc: r.ShouldAddWorkStats,
	}

	go func() {
		AddWorkerCount(1)
		defer AddWorkerCount(-1)

		TestWg.Add(1)
		defer TestWg.Done()

		worker.Run()

		r.removeWorker(worker.ID)
	}()

	return worker
}

func (r *Runner) removeWorker(id int) {
	r.workersLock.Lock()
	defer r.workersLock.Unlock()

	delete(r.workers, id)
}

// UpdateNumWorkers checks if the current number of workers is reasonable,
// and adds more if needed
func (r *Runner) UpdateNumWorkers(numChecks int64) {
	if r.isStaticWorkerCount {
		log.Warnf("Attempted to change runner %ds static worker count. Ignoring.", r.id)
		return
	}

	// Find which range the number of checks we're running falls in
	var desiredNumWorkers int
	switch {
	case numChecks <= 10:
		desiredNumWorkers = 4
	case numChecks <= 15:
		desiredNumWorkers = 10
	case numChecks <= 20:
		desiredNumWorkers = 15
	case numChecks <= 25:
		desiredNumWorkers = 20
	default:
		desiredNumWorkers = config.MaxNumWorkers
	}

	r.ensureMinWorkers(desiredNumWorkers)
}

// Stop closes the pending channel so all workers will exit their loop and terminate
// All publishers to the pending channel need to have stopped before Stop is called
func (r *Runner) Stop() {
	if !atomic.CompareAndSwapUint32(&r.isRunning, 1, 0) {
		log.Debugf("Runner %d already stopped, nothing to do here...", r.id)
		return
	}

	log.Infof("Runner %d is shutting down...", r.id)
	close(r.pendingCheckChan)

	// Stop checks that are still running
	globalDone := make(chan struct{})
	wg := sync.WaitGroup{}

	// Stop running checks
	r.checksTracker.WithRunningChecks(func(runningChecks map[check.ID]check.Check) {
		// Stop all python subprocesses
		terminateChecksRunningProcesses()

		for _, c := range runningChecks {
			wg.Add(1)
			go func(c check.Check) {
				log.Infof("Stopping check %v that is still running...", c)
				done := make(chan struct{})
				go func() {
					c.Stop()
					close(done)
					wg.Done()
				}()

				select {
				case <-done:
					// all good
				case <-time.After(stopCheckTimeout):
					// check is not responding
					log.Warnf("Check %v not responding after %v", c, stopCheckTimeout)
				}
			}(c)
		}
	})

	go func() {
		log.Debugf("Runner %d waiting for all the workers to exit...", r.id)
		wg.Wait()

		log.Debugf("All runner %d workers have been shut down", r.id)
		close(globalDone)
	}()

	select {
	case <-globalDone:
		log.Infof("Runner %d shut down", r.id)
	case <-time.After(stopAllChecksTimeout):
		log.Errorf(
			"Some checks on runner %d not responding after %v, timing out...",
			r.id,
			stopAllChecksTimeout,
		)
	}
}

// GetChan returns a write-only version of the pending channel
func (r *Runner) GetChan() chan<- check.Check {
	return r.pendingCheckChan
}

// SetScheduler sets the scheduler for the runner
func (r *Runner) SetScheduler(s *scheduler.Scheduler) {
	r.schedulerLock.Lock()
	r.scheduler = s
	r.schedulerLock.Unlock()
}

// ShouldAddWorkStats returns true if check stats should be preserved or not
func (r *Runner) ShouldAddWorkStats(id check.ID) bool {
	r.schedulerLock.Lock()
	defer r.schedulerLock.Unlock()

	if r.scheduler == nil {
		return true
	}

	if r.scheduler.IsCheckScheduled(id) {
		return true
	}

	return false
}

// StopCheck invokes the `Stop` method on a check if it's running. If the check
// is not running, this is a noop
func (r *Runner) StopCheck(id check.ID) error {
	done := make(chan bool)

	r.checksTracker.Lock()
	defer r.checksTracker.Unlock()

	if c, isRunning := r.checksTracker.Check(id); isRunning {
		log.Debugf("Stopping check %s", c.ID())
		go func() {
			// Remember that the check was stopped so that even if it runs we can discard its stats
			c.Stop()
			close(done)
		}()
	} else {
		log.Debugf("Check %s is not running, not stopping it", id)
		return nil
	}

	select {
	case <-done:
		return nil
	case <-time.After(stopCheckTimeout):
		return fmt.Errorf("timeout during stop operation on check id %s", id)
	}
}
