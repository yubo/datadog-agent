package util

import (
	"bufio"
	"bytes"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"

	"github.com/DataDog/datadog-agent/pkg/process/procutil"
	"github.com/DataDog/gopsutil/cpu"
	"github.com/DataDog/gopsutil/process"

	"github.com/DataDog/datadog-agent/pkg/config"
	"github.com/DataDog/datadog-agent/pkg/util/docker"
)

// ErrNotImplemented is the "not implemented" error given by `gopsutil` when an
// OS doesn't support and API. Unfortunately it's in an internal package so
// we can't import it so we'll copy it here.
var ErrNotImplemented = errors.New("not implemented yet")

// ReadLines reads contents from a file and splits them by new lines.
func ReadLines(filename string) ([]string, error) {
	f, err := os.Open(filename)
	if err != nil {
		return []string{""}, err
	}
	defer f.Close()

	var ret []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		ret = append(ret, scanner.Text())
	}
	return ret, scanner.Err()
}

// GetEnv retrieves the environment variable key. If it does not exist it returns the default.
func GetEnv(key string, dfault string, combineWith ...string) string {
	value := os.Getenv(key)
	if value == "" {
		value = dfault
	}

	switch len(combineWith) {
	case 0:
		return value
	case 1:
		return filepath.Join(value, combineWith[0])
	default:
		var b bytes.Buffer
		b.WriteString(value)
		for _, v := range combineWith {
			b.WriteRune('/')
			b.WriteString(v)
		}
		return b.String()
	}
}

// HostProc returns the location of a host's procfs. This can and will be
// overridden when running inside a container.
func HostProc(combineWith ...string) string {
	return GetEnv("HOST_PROC", "/proc", combineWith...)
}

// HostSys returns the location of a host's /sys. This can and will be overridden
// when running inside a container.
func HostSys(combineWith ...string) string {
	return GetEnv("HOST_SYS", "/sys", combineWith...)
}

// PathExists returns a boolean indicating if the given path exists on the file system.
func PathExists(filename string) bool {
	if _, err := os.Stat(filename); err == nil {
		return true
	}
	return false
}

// StringInSlice returns true if the given searchString is in the given slice, false otherwise.
func StringInSlice(slice []string, searchString string) bool {
	for _, curString := range slice {
		if curString == searchString {
			return true
		}
	}
	return false
}

// GetDockerSocketPath is only for exposing the sockpath out of the module
func GetDockerSocketPath() (string, error) {
	// If we don't have a docker.sock then return a known error.
	sockPath := GetEnv("DOCKER_SOCKET_PATH", "/var/run/docker.sock")
	if !PathExists(sockPath) {
		return "", docker.ErrDockerNotAvailable
	}
	return sockPath, nil
}

// GetProcRoot retrieves the current procfs dir we should use
func GetProcRoot() string {
	if v := os.Getenv("HOST_PROC"); v != "" {
		return v
	}

	if config.IsContainerized() && PathExists("/host") {
		return "/host/proc"
	}

	return "/proc"
}

// WithAllProcs will execute `fn` for every pid under procRoot. `fn` is
// passed the `pid`. If `fn` returns an error the iteration aborts,
// returning the last error returned from `fn`.
func WithAllProcs(procRoot string, fn func(int) error) error {
	files, err := ioutil.ReadDir(procRoot)
	if err != nil {
		return err
	}

	for _, f := range files {
		if !f.IsDir() || f.Name() == "." || f.Name() == ".." {
			continue
		}

		var pid int
		if pid, err = strconv.Atoi(f.Name()); err != nil {
			continue
		}

		if err = fn(pid); err != nil {
			return err
		}
	}

	return nil
}

// ConvertCPUStat converts gopsutil TimeStat object to CPUTimesStat in procutil
func ConvertCPUStat(s cpu.TimesStat) *procutil.CPUTimesStat {
	return &procutil.CPUTimesStat{
		User:      s.User,
		System:    s.System,
		Idle:      s.Idle,
		Nice:      s.Nice,
		Iowait:    s.Iowait,
		Irq:       s.Irq,
		Softirq:   s.Softirq,
		Steal:     s.Steal,
		Guest:     s.Guest,
		GuestNice: s.GuestNice,
		Stolen:    s.Stolen,
		Timestamp: s.Timestamp,
	}
}

// ConvertMemInfo converts gopsutil MemoryInfoStat object to MemoryInfoStat in procutil
func ConvertMemInfo(s *process.MemoryInfoStat) *procutil.MemoryInfoStat {
	return &procutil.MemoryInfoStat{
		RSS:  s.RSS,
		VMS:  s.VMS,
		Swap: s.Swap,
	}
}

// ConvertMemInfoEx converts gopsutil MemoryInfoExStat object to MemoryInfoExStat in procutil
func ConvertMemInfoEx(s *process.MemoryInfoExStat) *procutil.MemoryInfoExStat {
	return &procutil.MemoryInfoExStat{
		RSS:    s.RSS,
		VMS:    s.VMS,
		Shared: s.Shared,
		Text:   s.Text,
		Lib:    s.Lib,
		Data:   s.Data,
		Dirty:  s.Dirty,
	}
}

// ConvertIOStats converts gopsutil IOCounterStat object to IOCountersStat in procutil
func ConvertIOStats(s *process.IOCountersStat) *procutil.IOCountersStat {
	return &procutil.IOCountersStat{
		ReadCount:  s.ReadCount,
		WriteCount: s.WriteCount,
		ReadBytes:  s.ReadBytes,
		WriteBytes: s.WriteBytes,
	}
}

// ConvertCtxSwitches converts gopsutil NumCtxSwitchesStat object to NumCtxSwitchesStat in procutil
func ConvertCtxSwitches(s *process.NumCtxSwitchesStat) *procutil.NumCtxSwitchesStat {
	return &procutil.NumCtxSwitchesStat{
		Voluntary:   s.Voluntary,
		Involuntary: s.Involuntary,
	}
}
