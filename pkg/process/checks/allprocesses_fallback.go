// +build !windows,!linux

package checks

import (
	"github.com/DataDog/datadog-agent/pkg/process/net"
	"github.com/DataDog/datadog-agent/pkg/process/procutil"
	"github.com/DataDog/gopsutil/process"
)

func getAllProcesses(probe *procutil.Probe, pu *net.RemoteSysProbeUtil) (map[int32]*process.FilledProcess, error) {
	return process.AllProcesses()
}

func getAllProcStats(probe *procutil.Probe, pu *net.RemoteSysProbeUtil, pids []int32) (map[int32]*process.FilledProcess, error) {
	return getAllProcesses(probe)
}
