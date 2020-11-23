// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2020 Datadog, Inc.

// +build functionaltests

package tests

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"time"

	"github.com/pkg/errors"
)

type testDriveOverlay struct {
	testDrive        *testDrive
	lowerLayersCount int
	lowerLayersPaths map[int]string
	overlayPaths     map[string]string
}

func newTestDriveOverlay(lowerLayersCount int) (*testDriveOverlay, error) {

	lowerLayersPaths := make(map[int]string)
	overlayPaths := make(map[string]string)

	// Check the lowerLayersCount value
	if lowerLayersCount < 1 {
		return nil, errors.New("lowerLayerName must be greater than 0")
	}

	// Create the loopback drive that will store the Overlay FS
	testDrive, err := newTestDrive("ext4", []string{})
	if err != nil {
		return nil, err
	}

	// Create lower layers
	for lowerID := 0; lowerID < lowerLayersCount; lowerID++ {
		lowerLayerName := fmt.Sprintf("/lower_%0.2d", lowerID)
		directoryName := path.Join(testDrive.mountPoint, lowerLayerName)
		err := os.Mkdir(directoryName, 0755)
		if err != nil {
			testDrive.Close()
			return nil, errors.Wrapf(err, "failed to create %s", lowerLayerName)
		}
		lowerLayersPaths[lowerID] = directoryName
	}

	// Create upper, workdir and merged directories
	for _, layerName := range []string{"upper", "workdir", "merged"} {
		directoryName := path.Join(testDrive.mountPoint, layerName)
		err := os.Mkdir(directoryName, 0755)
		if err != nil {
			testDrive.Close()
			return nil, errors.Wrapf(err, "failed to create %s", layerName)
		}
		overlayPaths[layerName] = directoryName
	}

	// Mount the Overlay FS
	lowerLayersJoined := lowerLayersPaths[0]
	for i := 1; i < len(lowerLayersPaths); i++ {
		lowerLayersJoined += fmt.Sprintf(":%s", lowerLayersPaths[i])
	}
	overlayOptions := fmt.Sprintf("lowerdir=%s,upperdir=%s,workdir=%s", lowerLayersJoined, overlayPaths["upper"], overlayPaths["workdir"])
	mountCmd := exec.Command("mount", "-t", "overlay", "overlay", "-o", overlayOptions, overlayPaths["merged"])
	fmt.Printf("CMD %s\n", mountCmd.String())
	if err := mountCmd.Run(); err != nil {
		testDrive.Close()
		return nil, errors.Wrap(err, "failed to mount overlay")
	}

	return &testDriveOverlay{
		testDrive:        testDrive,
		lowerLayersCount: lowerLayersCount,
		lowerLayersPaths: lowerLayersPaths,
		overlayPaths:     overlayPaths,
	}, nil
}

func (tdo *testDriveOverlay) Close() {
	// Unmount the Overlay FS
	unmountCmd := exec.Command("umount", "-f", tdo.overlayPaths["merged"])
	fmt.Printf("CMD %s\n", unmountCmd.String())
	if err := unmountCmd.Run(); err != nil {
		fmt.Println(err)
	}

	// Unmount the loopback drive
	time.Sleep(time.Second)
	tdo.testDrive.Close()
}
