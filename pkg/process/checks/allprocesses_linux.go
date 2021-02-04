// +build linux

package checks

import (
	"time"

	model "github.com/DataDog/agent-payload/process"
	"github.com/DataDog/datadog-agent/pkg/process/net"
	"github.com/DataDog/datadog-agent/pkg/process/procutil"
	"github.com/DataDog/gopsutil/process"
)

// getAllProcesses uses a procutil.Probe and system-probe to fetch processes,
// then convert them into FilledProcesses for compatibility.
// if system-probe client is not nil, merge procutil data and system-probe,
// otherwise uses procutil to collect with best effort(fields with extra permission might not be available)
func getAllProcesses(probe *procutil.Probe, pu *net.RemoteSysProbeUtil) (map[int32]*process.FilledProcess, error) {
	var procs map[int32]*procutil.Process
	var err error

	if pu != nil {
		procs, err = probe.ProcessesByPIDWithoutPerm(time.Now())
		if err != nil {
			return nil, err
		}

		// this is also best effort, if system-probe query failed, just use what we have
		stats, err := pu.GetProcStats()
		if err == nil {
			procs = mergeProcWithStats(procs, stats)
		}
	} else {
		procs, err = probe.ProcessesByPIDWithPerm(time.Now())
		if err != nil {
			return nil, err
		}
	}

	return procutil.ConvertAllProcesses(procs), nil
}

func getAllProcStats(probe *procutil.Probe, pids []int32) (map[int32]*process.FilledProcess, error) {
	stats, err := probe.StatsForPIDsWithPerm(pids, time.Now())
	if err != nil {
		return nil, err
	}

	procs := make(map[int32]*process.FilledProcess, len(stats))
	for pid, stat := range stats {
		procs[pid] = procutil.ConvertToFilledProcess(&procutil.Process{Pid: pid, Stats: stat})
	}
	return procs, nil
}

// mergeProcWithStats takes a process by PID map and fill the stats into the processes in the map
func mergeProcWithStats(procs map[int32]*procutil.Process, stats *model.ProcStatsWithPermByPID) map[int32]*procutil.Process {
	for pid, proc := range procs {
		if s, ok := stats.StatsByPID[pid]; ok {
			proc.Stats.OpenFdCount = s.OpenFDCount
			proc.Stats.IOStat.ReadCount = s.ReadCount
			proc.Stats.IOStat.WriteCount = s.WriteCount
			proc.Stats.IOStat.ReadBytes = s.ReadBytes
			proc.Stats.IOStat.WriteBytes = s.WriteBytes
		}
	}
	return procs
}
