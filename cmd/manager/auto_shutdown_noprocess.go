// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package manager

import (
	"regexp"

	"github.com/DataDog/datadog-agent/pkg/util/log"
	"github.com/DataDog/gopsutil/process"
)

type processes map[int32]*process.FilledProcess

var (
	defaultRegexps = []*regexp.Regexp{
		regexp.MustCompile("pause|s6-svscan|s6-supervise"),
		regexp.MustCompile("agent|process-agent|trace-agent|security-agent|system-probe"),
	}
	processFetcher = fetchProcesses
)

func fetchProcesses() (processes, error) {
	return process.AllProcesses()
}

type noProcessShutdown struct {
	excludeProcesses []*regexp.Regexp
}

// NoProcessShutdown creates a shutdown detector based on running processes
func NoProcessShutdown(r []*regexp.Regexp) ShutdownDetector {
	return &noProcessShutdown{excludeProcesses: r}
}

// DefaultNoProcessShutdown creates the default NoProcess shutdown detector
func DefaultNoProcessShutdown() ShutdownDetector {
	return NoProcessShutdown(defaultRegexps)
}

func (s *noProcessShutdown) check() bool {
	processes, err := processFetcher()
	if err != nil {
		log.Debugf("Unable to get processes list to trigger autoshutdown, err: %w", err)
		return false
	}

	for pid, p := range processes {
		for _, r := range s.excludeProcesses {
			if matched := r.MatchString(p.Name); matched {
				delete(processes, pid)
			}
		}
	}

	log.Debugf("Processes preventing shutdown: p: %v", processes)
	return len(processes) == 0
}
