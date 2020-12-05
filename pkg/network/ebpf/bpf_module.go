// +build linux_bpf

package ebpf

import (
	"fmt"
	"io"
	"os"
	"path"
)

// ReadBPFModule from the asset file
func ReadBPFModule(bpfDir string, debug bool) (io.ReaderAt, error) {
	file := "tracer.o"
	if debug {
		file = "tracer-debug.o"
	}

	elfPath := path.Join(bpfDir, file)
	ebpfReader, err := os.Open(elfPath)
	if err != nil {
		return nil, fmt.Errorf("couldn't find asset: %s", err)
	}

	return ebpfReader, nil
}

// ReadOffsetBPFModule from the asset file
func ReadOffsetBPFModule(bpfDir string, debug bool) (io.ReaderAt, error) {
	file := "offset-guess.o"
	if debug {
		file = "offset-guess-debug.o"
	}

	elfPath := path.Join(bpfDir, file)
	ebpfReader, err := os.Open(elfPath)
	if err != nil {
		return nil, fmt.Errorf("couldn't find asset: %s", err)
	}

	return ebpfReader, nil
}
