// +build !windows

package main

import (
	_ "net/http/pprof"

	"github.com/DataDog/datadog-agent/cmd/process-agent/app"
)

func main() {
	app.Run()
}
