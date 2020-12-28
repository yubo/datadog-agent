package checks

import (
	"time"

	"github.com/DataDog/datadog-agent/pkg/process/procutil"

	"github.com/DataDog/datadog-agent/pkg/util/containers"
	"github.com/DataDog/gopsutil/cpu"

	model "github.com/DataDog/agent-payload/process"
	"github.com/DataDog/datadog-agent/pkg/process/config"
	"github.com/DataDog/datadog-agent/pkg/process/util"
)

// RTProcess is a singleton RTProcessCheck.
var RTProcess = &RTProcessCheck{}

// RTProcessCheck collects numeric statistics about the live processes.
// The instance stores state between checks for calculation of rates and CPU.
type RTProcessCheck struct {
	probe *procutil.Probe

	sysInfo      *model.SystemInfo
	lastCPUTime  *procutil.CPUTimesStat
	lastStats    map[int32]*procutil.Stats
	lastCtrRates map[string]util.ContainerRateMetrics
	lastRun      time.Time

	// RTProcessCheck needs to know the PIDs that ProcessCheck is collected,
	// so we will keep a reference for that check in order to get PIDs
	processCheck *ProcessCheck
}

// Init initializes a new RTProcessCheck instance.
func (r *RTProcessCheck) Init(_ *config.AgentConfig, info *model.SystemInfo) {
	r.sysInfo = info
}

// Name returns the name of the RTProcessCheck.
func (r *RTProcessCheck) Name() string { return "rtprocess" }

// RealTime indicates if this check only runs in real-time mode.
func (r *RTProcessCheck) RealTime() bool { return true }

// Run runs the RTProcessCheck to collect statistics about the running processes.
// On most POSIX systems these statistics are collected from procfs. The bulk
// of this collection is abstracted into the `gopsutil` library.
// Processes are split up into a chunks of at most 100 processes per message to
// limit the message size on intake.
// See agent.proto for the schema of the message and models used.
func (r *RTProcessCheck) Run(cfg *config.AgentConfig, groupID int32) ([]model.MessageBody, error) {
	cpuTimes, err := cpu.Times(false)
	if err != nil {
		return nil, err
	}
	if len(cpuTimes) == 0 {
		return nil, errEmptyCPUTime
	}
	cpuTime := procutil.ConvertCPUStat(cpuTimes[0])

	stats, err := r.probe.StatsForPIDs(r.processCheck.GetLastPIDs(), time.Now())
	if err != nil {
		return nil, err
	}
	ctrList, _ := util.GetContainers()

	// End check early if this is our first run.
	if r.lastStats == nil {
		r.lastCtrRates = util.ExtractContainerRateMetric(ctrList)
		r.lastStats = stats
		r.lastCPUTime = cpuTime
		r.lastRun = time.Now()
		return nil, nil
	}

	chunkedStats := fmtProcessStats(cfg, stats, r.lastStats,
		ctrList, cpuTime, r.lastCPUTime, r.lastRun)
	groupSize := len(chunkedStats)
	chunkedCtrStats := fmtContainerStats(ctrList, r.lastCtrRates, r.lastRun, groupSize)
	messages := make([]model.MessageBody, 0, groupSize)
	for i := 0; i < groupSize; i++ {
		messages = append(messages, &model.CollectorRealTime{
			HostName:          cfg.HostName,
			Stats:             chunkedStats[i],
			ContainerStats:    chunkedCtrStats[i],
			GroupId:           groupID,
			GroupSize:         int32(groupSize),
			NumCpus:           int32(len(r.sysInfo.Cpus)),
			TotalMemory:       r.sysInfo.TotalMemory,
			ContainerHostType: cfg.ContainerHostType,
		})
	}

	// Store the last state for comparison on the next run.
	// Note: not storing the filtered in case there are new processes that haven't had a chance to show up twice.
	r.lastRun = time.Now()
	r.lastStats = stats
	r.lastCtrRates = util.ExtractContainerRateMetric(ctrList)
	r.lastCPUTime = cpuTime

	return messages, nil
}

// fmtProcessStats formats and chunks a slice of ProcessStat into chunks.
func fmtProcessStats(
	cfg *config.AgentConfig,
	stats, lastStats map[int32]*procutil.Stats,
	ctrList []*containers.Container,
	syst2, syst1 *procutil.CPUTimesStat,
	lastRun time.Time,
) [][]*model.ProcessStat {
	cidByPid := make(map[int32]string, len(ctrList))
	for _, c := range ctrList {
		for _, p := range c.Pids {
			cidByPid[p] = c.ID
		}
	}

	chunked := make([][]*model.ProcessStat, 0)
	chunk := make([]*model.ProcessStat, 0, cfg.MaxPerMessage)
	for pid, st := range stats {
		// NOTE: in ProcessCheck the processes could be skipped if the command line matches blacklist,
		// we don't do it here so we might send stats for blacklisted processes, but they won't be processed
		// by the backend system so it's safe
		chunk = append(chunk, &model.ProcessStat{
			Pid:         pid,
			CreateTime:  st.CreateTime,
			Memory:      formatMemory(st),
			Cpu:         formatCPU(st, lastStats[pid], syst2, syst1),
			Nice:        st.Nice,
			Threads:     st.NumThreads,
			OpenFdCount: st.OpenFdCount,
			// what should we do here?
			// ProcessState:           model.ProcessState(model.ProcessState_value[st.Status]),
			IoStat:                 formatIO(st.IOStat, lastStats[pid].IOStat, lastRun),
			VoluntaryCtxSwitches:   uint64(st.CtxSwitches.Voluntary),
			InvoluntaryCtxSwitches: uint64(st.CtxSwitches.Involuntary),
			ContainerId:            cidByPid[pid],
		})
		if len(chunk) == cfg.MaxPerMessage {
			chunked = append(chunked, chunk)
			chunk = make([]*model.ProcessStat, 0, cfg.MaxPerMessage)
		}
	}
	if len(chunk) > 0 {
		chunked = append(chunked, chunk)
	}
	return chunked
}

func calculateRate(cur, prev uint64, before time.Time) float32 {
	now := time.Now()
	diff := now.Unix() - before.Unix()
	if before.IsZero() || diff <= 0 || prev == 0 || prev > cur {
		return 0
	}
	return float32(cur-prev) / float32(diff)
}
