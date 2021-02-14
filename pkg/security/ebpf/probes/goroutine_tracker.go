// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2020 Datadog, Inc.

// +build linux

package probes

import "github.com/DataDog/ebpf/manager"

// goroutineTrackerProbes holds the list of probes used to track goroutines
var goroutineTrackerProbes = []*manager.Probe{
	{
		UID:     SecurityAgentUID,
		Section: GetGoroutineTrackerSection(),
	},
}

// GetGoroutineTrackerSection returns the goroutine tracker program section
func GetGoroutineTrackerSection() string {
	return "uprobe/runtime.execute"
}

func getGoroutineTracker() []*manager.Probe {
	return goroutineTrackerProbes
}
