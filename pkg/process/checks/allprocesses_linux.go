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
			procs = mergeProcWithSysprobeStats(procs, stats)
		}
	} else {
		procs, err = probe.ProcessesByPIDWithPerm(time.Now())
		if err != nil {
			return nil, err
		}
	}

	return procutil.ConvertAllProcesses(procs), nil
}

func getAllProcStats(probe *procutil.Probe, pu *net.RemoteSysProbeUtil, pids []int32) (map[int32]*process.FilledProcess, error) {
	var stats map[int32]*procutil.Stats
	var err error

	if pu != nil {
		stats, err = probe.StatsForPIDsWithoutPerm(pids, time.Now())
		if err != nil {
			return nil, err
		}
		// this is also best effort, if system-probe query failed, just use what we have
		pStats, err := pu.GetProcStats()
		if err == nil {
			stats = mergeStatWithSysprobeStats(stats, pStats)
		}
	} else {
		stats, err = probe.StatsForPIDsWithPerm(pids, time.Now())
		if err != nil {
			return nil, err
		}
	}

	procs := make(map[int32]*process.FilledProcess, len(stats))
	for pid, stat := range stats {
		procs[pid] = procutil.ConvertToFilledProcess(&procutil.Process{Pid: pid, Stats: stat})
	}
	return procs, nil
}

// mergeProcWithSysprobeStats takes a process by PID map and fill the stats from system probe into the processes in the map
func mergeProcWithSysprobeStats(procs map[int32]*procutil.Process, pStats *model.ProcStatsWithPermByPID) map[int32]*procutil.Process {
	for pid, proc := range procs {
		if s, ok := pStats.StatsByPID[pid]; ok {
			proc.Stats.OpenFdCount = s.OpenFDCount
			proc.Stats.IOStat.ReadCount = s.ReadCount
			proc.Stats.IOStat.WriteCount = s.WriteCount
			proc.Stats.IOStat.ReadBytes = s.ReadBytes
			proc.Stats.IOStat.WriteBytes = s.WriteBytes
		}
	}
	return procs
}

// mergeStatWithSysprobeStats takes a stats by PID map and fill the stats from system probe into the stats in the map
func mergeStatWithSysprobeStats(stats map[int32]*procutil.Stats, pStats *model.ProcStatsWithPermByPID) map[int32]*procutil.Stats {
	for pid, stat := range stats {
		if s, ok := pStats.StatsByPID[pid]; ok {
			stat.OpenFdCount = s.OpenFDCount
			stat.IOStat.ReadCount = s.ReadCount
			stat.IOStat.WriteCount = s.WriteCount
			stat.IOStat.ReadBytes = s.ReadBytes
			stat.IOStat.WriteBytes = s.WriteBytes
		}
	}
	return stats
}
