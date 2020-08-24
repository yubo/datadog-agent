// +build linux_bpf

package compiler

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/DataDog/datadog-agent/pkg/ebpf/bytecode"
	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"
)

func TestCompilerMatch(t *testing.T) {
	c, err := NewEBPFCompiler(true)
	require.NoError(t, err)
	defer c.Close()
	t.Logf("flags: %+v\n", c.defaultCflags)

	onDiskFilename := "../c/tracer-ebpf-static.o"
	err = c.CompileToObjectFile("../c/tracer-ebpf.c", onDiskFilename)
	require.NoError(t, err)

	bs, err := ioutil.ReadFile(onDiskFilename)
	require.NoError(t, err)

	bundleFilename := "pkg/ebpf/c/tracer-ebpf.o"
	actualReader, err := bytecode.GetReader("", bundleFilename)
	require.NoError(t, err)

	actual, err := ioutil.ReadAll(actualReader)
	require.NoError(t, err)

	assert.True(t, bytes.Equal(bs, actual), fmt.Sprintf("on-disk file %s and statically-linked clang compiled content %s are different", onDiskFilename, bundleFilename))
}
