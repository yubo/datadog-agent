// +build !windows

package procutil

import (
	"github.com/DataDog/gopsutil/process"
)

// ProcessProbe is used to parse /proc file system and extract process information
type ProcessProbe struct{}

// NewProcessProbe instantiates a new ProcessProbe
func NewProcessProbe() *ProcessProbe {
	return &ProcessProbe{}
}

// ProcessByPID parses procfs and returns a map that holds processes by PIDs
func (p *ProcessProbe) ProcessesByPID() (map[int32]*Process, error) {
	rawProcesses, err := process.AllProcesses()
	if err != nil {
		return nil, err
	}

	procByPID := make(map[int32]*Process)
	for pid, proc := range rawProcesses {
		procByPID[pid] = &Process{
			Pid:      proc.Pid,
			Ppid:     proc.Ppid,
			NsPid:    proc.NsPid,
			Status:   proc.Status,
			Name:     proc.Name,
			Cwd:      proc.Cwd,
			Exe:      proc.Exe,
			Cmdline:  proc.Cmdline,
			Username: proc.Username,
			Uids:     proc.Uids,
			Gids:     proc.Gids,
			Stats: &Stats{
				CreateTime:  proc.CreateTime,
				Nice:        proc.Nice,
				OpenFdCount: proc.OpenFdCount,
				NumThreads:  proc.NumThreads,
				CPUTime:     AssignCPUStat(proc.CpuTime),
				MemInfo:     AssignMemInfo(proc.MemInfo),
				MemInfoEx:   AssignMemInfoEx(proc.MemInfoEx),
				IOStat:      AssignIOStats(proc.IOStat),
				CtxSwitches: AssignCtxSwitches(proc.CtxSwitches),
			},
		}
	}
	return procByPID, nil
}
