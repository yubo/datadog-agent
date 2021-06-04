// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

// +build linux

package utils

import "golang.org/x/sys/unix"

// NumCPU returns the count of CPUs in the CPU affinity mask of the pid 1 process
func NumCPU() (int, error) {
	cpuSet := unix.CPUSet{}
	if err := unix.SchedGetaffinity(1, &cpuSet); err != nil {
		return 0, err
	}
	return cpuSet.Count(), nil
}
