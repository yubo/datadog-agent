// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package manager

import (
	"fmt"
	"os"
	"time"

	"github.com/DataDog/datadog-agent/pkg/config"
	"github.com/DataDog/datadog-agent/pkg/util/log"

	"golang.org/x/net/context"
)

const (
	defaultShutdownTicker = 30 * time.Second
)

// ShutdownDetector is common interface for shutdown mechanisms
type ShutdownDetector interface {
	check() bool
}

var shutdownRegistry = map[string]ShutdownDetector{
	"noprocess": DefaultNoProcessShutdown(),
}

// ConfigureAutoShutdown starts automatic shutdown mechanism if necessary
func ConfigureAutoShutdown(ctx context.Context) error {
	if method := config.Datadog.GetString("auto_shutdown_method"); method != "" {
		validationPeriod := time.Duration(config.Datadog.GetInt("auto_shutdown_validation_period")) * time.Second
		if sd, found := shutdownRegistry[method]; found {
			return startAutoShutdown(ctx, sd, defaultShutdownTicker, validationPeriod)
		}
	}

	return nil
}

func startAutoShutdown(ctx context.Context, sd ShutdownDetector, tickerPeriod, validationPeriod time.Duration) error {
	if sd == nil {
		return fmt.Errorf("a shutdown detector must be provided")
	}

	selfProcess, err := os.FindProcess(os.Getpid())
	if err != nil {
		return fmt.Errorf("cannot find own process, err: %w", err)
	}

	log.Info("Starting auto-shutdown watcher")
	lastSeen := time.Now()
	ticker := time.NewTicker(tickerPeriod)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return

			case <-ticker.C:
				shutdownConditionFound := sd.check()
				if shutdownConditionFound {
					if lastSeen.Add(validationPeriod).Before(time.Now()) {
						log.Info("Conditions met for automatic shutdown: triggering stop sequence")
						if err := selfProcess.Signal(os.Interrupt); err != nil {
							log.Errorf("Unable to send termination signal - will use os.exit, err: %v", err)
							os.Exit(1)
						}
						return
					}
				} else {
					lastSeen = time.Now()
				}
			}
		}
	}()

	return nil
}
