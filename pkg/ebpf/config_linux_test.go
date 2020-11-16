// +build linux_bpf

package ebpf

import (
	"runtime"
	"strings"
	"testing"

	"github.com/DataDog/datadog-agent/pkg/ebpf/bytecode"
	"github.com/DataDog/ebpf/manager"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChooseSyscall(t *testing.T) {
	c := NewDefaultConfig()

	_, err := c.chooseSyscallProbe("wrongformat", "", "")
	assert.Error(t, err)

	_, err = c.chooseSyscallProbe("nontracepoint/what/wrong", "", "")
	assert.Error(t, err)

	_, err = c.chooseSyscallProbe(bytecode.TraceSysBindEnter, "", "wrongformat")
	assert.Error(t, err)

	// kprobe syscalls must match
	_, err = c.chooseSyscallProbe(bytecode.TraceSysBindEnter, bytecode.SysBindIndirect, bytecode.SysSocket)
	assert.Error(t, err)

	tp, err := c.chooseSyscallProbe(bytecode.TraceSysBindEnter, bytecode.SysBindIndirect, bytecode.SysBind)
	require.NoError(t, err)

	fnName, err := manager.GetSyscallFnName("sys_bind")
	require.NoError(t, err)

	// intentionally leaving amd64/arm64 explicit to ensure they are included in the prefix map
	switch runtime.GOARCH {
	case "amd64":
		if strings.HasPrefix(fnName, indirectSyscallPrefixes[runtime.GOARCH]) {
			assert.Equal(t, bytecode.SysBindIndirect, tp)
		} else {
			assert.Equal(t, bytecode.SysBind, tp)
		}
	case "arm64":
		if strings.HasPrefix(fnName, indirectSyscallPrefixes[runtime.GOARCH]) {
			assert.Equal(t, bytecode.SysBindIndirect, tp)
		} else {
			assert.Equal(t, bytecode.SysBind, tp)
		}
	default:
		assert.Equal(t, bytecode.SysBind, tp)
	}

	c.EnableTracepoints = true
	tp, err = c.chooseSyscallProbe(bytecode.TraceSysBindEnter, bytecode.SysBindIndirect, bytecode.SysBind)
	require.NoError(t, err)

	assert.Equal(t, bytecode.TraceSysBindEnter, tp)
}
