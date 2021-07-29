// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package runner

import (
	"context"
	"fmt"
	"time"

	"github.com/DataDog/datadog-agent/pkg/aggregator"
	"github.com/DataDog/datadog-agent/pkg/collector/check"
	"github.com/DataDog/datadog-agent/pkg/config"
	"github.com/DataDog/datadog-agent/pkg/metrics"
	"github.com/DataDog/datadog-agent/pkg/util"
	"github.com/DataDog/datadog-agent/pkg/util/log"
)

const (
	// How long is the first series of check runs we want to log
	firstRunSeries uint64 = 5
)

// Worker is an object that encapsulates the logic to manage a loop of processing
// checks over the provided `PendingCheckChan`
type Worker struct {
	ID                     int
	ChecksTracker          *runningChecksTracker
	PendingCheckChan       chan check.Check
	RunnerID               int
	ShouldAddWorkStatsFunc func(id check.ID) bool
}

type timeVar time.Time

func (tv timeVar) String() string { return fmt.Sprintf("\"%s\"", time.Time(tv).Format(time.RFC3339)) }

// Run waits for checks and run them as long as they arrive on the channel
func (w *Worker) Run() {
	log.Debugf("Runner %d, worker %d: Ready to process checks...", w.RunnerID, w.ID)

	for check := range w.PendingCheckChan {
		// Add check to tracker if it's not already running
		if !w.ChecksTracker.AddCheck(check) {
			log.Debugf("Check %s is already running, skip execution...", check)
			continue
		}

		AddRunningCheckCount(1)

		doLog, lastLog := w.shouldLog(check.ID())

		if doLog {
			log.Infoc("Running check", "check", check)
		} else {
			log.Debugc("Running check", "check", check)
		}

		// run the check
		var checkErr error
		t0 := time.Now()

		SetRunningStats(check.ID(), timeVar(t0))
		checkErr = check.Run()
		DeleteRunningStats(check.ID())

		longRunning := check.Interval() == 0

		checkWarnings := check.GetWarnings()

		// use the default sender for the service checks
		sender, err := aggregator.GetDefaultSender()
		if err != nil {
			log.Errorf("Error getting default sender: %v. Not sending status check for %s", err, check)
		}
		serviceCheckTags := []string{fmt.Sprintf("check:%s", check.String())}
		serviceCheckStatus := metrics.ServiceCheckOK

		hostname, _ := util.GetHostname(context.TODO())

		if len(checkWarnings) != 0 {
			AddWarningsCount(len(checkWarnings))
			serviceCheckStatus = metrics.ServiceCheckWarning
		}

		if checkErr != nil {
			log.Errorf("Error running check %s: %s", check, checkErr)
			AddErrorsCount(1)
			serviceCheckStatus = metrics.ServiceCheckCritical
		}

		if sender != nil && !longRunning {
			sender.ServiceCheck("datadog.agent.check_status", serviceCheckStatus, hostname, serviceCheckTags, "")
			sender.Commit()
		}

		// remove the check from the running list
		w.ChecksTracker.DeleteCheck(check.ID())

		// publish statistics about this run
		AddRunningCheckCount(-1)
		AddRunsCount(1)

		if !longRunning || len(checkWarnings) != 0 || checkErr != nil {
			// If the scheduler isn't assigned (it should), just add stats
			// otherwise only do so if the check is in the scheduler
			if w.ShouldAddWorkStatsFunc(check.ID()) {
				sStats, _ := check.GetSenderStats()
				AddWorkStats(check, time.Since(t0), checkErr, checkWarnings, sStats)
			}
		}

		l := "Done running check"
		if doLog {
			if lastLog {
				l = l + fmt.Sprintf(", next runs will be logged every %v runs", config.Datadog.GetInt64("logging_frequency"))
			}
			log.Infoc(l, "check", check.String())
		} else {
			log.Debugc(l, "check", check.String())
		}

		if check.Interval() == 0 {
			log.Infof("Check %v one-time's execution has finished", check)
			return
		}
	}

	log.Debugf("Runner %d, worker %d: Finished processing checks.", w.RunnerID, w.ID)
}

func (w *Worker) shouldLog(id check.ID) (doLog bool, lastLog bool) {
	loggingFrequency := uint64(config.Datadog.GetInt64("logging_frequency"))

	// If this is the first time we see the check, log it
	stats, idFound := CheckStats(id)
	if !idFound {
		doLog = true
		lastLog = false
		return
	}

	// we log the first firstRunSeries times, then every loggingFrequency times
	doLog = stats.TotalRuns <= firstRunSeries || stats.TotalRuns%loggingFrequency == 0
	// we print a special message when we change logging frequency
	lastLog = stats.TotalRuns == firstRunSeries
	return
}
