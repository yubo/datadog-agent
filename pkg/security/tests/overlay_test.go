// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2020 Datadog, Inc.

// +build functionaltests

package tests

import (
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/DataDog/datadog-agent/pkg/security/rules"

	"github.com/pkg/errors"
)

func createFile(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return errors.Wrapf(err, "failed to create %s", filename)
	}
	defer file.Close()

	_, err = file.WriteString(filename)
	if err != nil {
		return errors.Wrapf(err, "failed to write to %s", filename)
	}

	return nil
}

func testOpenReadFile(test *testModule, filename string, expectedOverlayNumLower int32) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	f.Close()

	event, _, err := test.GetEvent()
	if err != nil {
		return err
	}
	fmt.Printf("Event: %v\n", event)
	if event.Open.OverlayNumLower != expectedOverlayNumLower {
		return errors.Errorf("expected OverlayNumLower %d, got %d", expectedOverlayNumLower, event.Open.OverlayNumLower)
	}
	return nil
}

func testOpenWriteFile(test *testModule, filename string, expectedOverlayNumLower int32) error {
	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE, 0400)
	if err != nil {
		return err
	}
	f.Close()

	event, _, err := test.GetEvent()
	if err != nil {
		return err
	}
	fmt.Printf("Event: %v\n", event)
	if event.Open.OverlayNumLower != expectedOverlayNumLower {
		return errors.Errorf("expected OverlayNumLower %d for %s, got %d", expectedOverlayNumLower, filename, event.Open.OverlayNumLower)
	}
	return nil
}

func TestOverlayReadLowerFromUpper(t *testing.T) {
	// Don't run the test inside Docker
	if testEnvironment == DockerEnvironment {
		t.Skip()

	}

	// Create the Overlay Drive with 3 lower layers
	testDriveOverlay, err := newTestDriveOverlay(3)
	if err != nil {
		t.Fatal(err)
	}
	defer testDriveOverlay.Close()

	// Create files in the three lower layers
	pathname := path.Join(testDriveOverlay.lowerLayersPaths[0], "0__.txt")
	fmt.Println(pathname)
	if err := createFile(pathname); err != nil {
		t.Fatal(err)
	}
	pathname = path.Join(testDriveOverlay.lowerLayersPaths[0], "0_2.txt")
	fmt.Println(pathname)
	if err := createFile(pathname); err != nil {
		t.Fatal(err)
	}
	pathname = path.Join(testDriveOverlay.lowerLayersPaths[2], "0_2.txt")
	fmt.Println(pathname)
	if err := createFile(pathname); err != nil {
		t.Fatal(err)
	}
	pathname = path.Join(testDriveOverlay.lowerLayersPaths[1], "_1_.txt")
	fmt.Println(pathname)
	if err := createFile(pathname); err != nil {
		t.Fatal(err)
	}
	pathname = path.Join(testDriveOverlay.lowerLayersPaths[2], "__2.txt")
	fmt.Println(pathname)
	if err := createFile(pathname); err != nil {
		t.Fatal(err)
	}

	// Open files for reading in lower layers
	rule := &rules.RuleDefinition{
		ID:         "overlay_lower_read_rule",
		Expression: `open.filename == "{{.Root}}/0__.txt" || open.filename == "{{.Root}}/0_2.txt" || open.filename == "{{.Root}}/_1_.txt" || open.filename == "{{.Root}}/__2.txt"`,
	}

	test, err := newTestModule(nil, []*rules.RuleDefinition{rule}, testOpts{testDir: testDriveOverlay.overlayPaths["merged"]})
	if err != nil {
		t.Fatal(err)
	}
	defer test.Close()

	t.Run("overlay_read_lower", func(t *testing.T) {
		var expectedOverlayNumLower int32 = 1
		if err := testOpenReadFile(test, path.Join(testDriveOverlay.overlayPaths["merged"], "0__.txt"), expectedOverlayNumLower); err != nil {
			t.Fatal(err)
		}
		if err := testOpenReadFile(test, path.Join(testDriveOverlay.overlayPaths["merged"], "0_2.txt"), expectedOverlayNumLower); err != nil {
			t.Fatal(err)
		}
		if err := testOpenReadFile(test, path.Join(testDriveOverlay.overlayPaths["merged"], "_1_.txt"), expectedOverlayNumLower); err != nil {
			t.Fatal(err)
		}
		if err := testOpenReadFile(test, path.Join(testDriveOverlay.overlayPaths["merged"], "__2.txt"), expectedOverlayNumLower); err != nil {
			t.Fatal(err)
		}
	})
}

func TestOverlayWriteLowerFromUpper(t *testing.T) {
	// Don't run the test inside Docker
	if testEnvironment == DockerEnvironment {
		t.Skip()

	}

	// Create the Overlay Drive with 3 lower layers
	testDriveOverlay, err := newTestDriveOverlay(3)
	if err != nil {
		t.Fatal(err)
	}
	defer testDriveOverlay.Close()

	// Create files in the three lower layers
	pathname := path.Join(testDriveOverlay.lowerLayersPaths[0], "0__.txt")
	fmt.Println(pathname)
	if err := createFile(pathname); err != nil {
		t.Fatal(err)
	}
	pathname = path.Join(testDriveOverlay.lowerLayersPaths[0], "0_2.txt")
	fmt.Println(pathname)
	if err := createFile(pathname); err != nil {
		t.Fatal(err)
	}
	pathname = path.Join(testDriveOverlay.lowerLayersPaths[2], "0_2.txt")
	fmt.Println(pathname)
	if err := createFile(pathname); err != nil {
		t.Fatal(err)
	}
	pathname = path.Join(testDriveOverlay.lowerLayersPaths[1], "_1_.txt")
	fmt.Println(pathname)
	if err := createFile(pathname); err != nil {
		t.Fatal(err)
	}
	pathname = path.Join(testDriveOverlay.lowerLayersPaths[2], "__2.txt")
	fmt.Println(pathname)
	if err := createFile(pathname); err != nil {
		t.Fatal(err)
	}

	// Open files for writing in the upper layer
	rule := &rules.RuleDefinition{
		ID:         "overlay_lower_write_rule",
		Expression: `open.filename == "{{.Root}}/0__.txt" || open.filename == "{{.Root}}/0_2.txt" || open.filename == "{{.Root}}/_1_.txt" || open.filename == "{{.Root}}/__2.txt"`,
	}

	test, err := newTestModule(nil, []*rules.RuleDefinition{rule}, testOpts{testDir: testDriveOverlay.overlayPaths["merged"]})
	if err != nil {
		t.Fatal(err)
	}
	defer test.Close()

	t.Run("overlay_write_lower", func(t *testing.T) {
		var expectedOverlayNumLower int32 = 0
		if err := testOpenWriteFile(test, path.Join(testDriveOverlay.overlayPaths["merged"], "0__.txt"), expectedOverlayNumLower); err != nil {
			t.Fatal(err)
		}
		if err := testOpenWriteFile(test, path.Join(testDriveOverlay.overlayPaths["merged"], "0_2.txt"), expectedOverlayNumLower); err != nil {
			t.Fatal(err)
		}
		if err := testOpenWriteFile(test, path.Join(testDriveOverlay.overlayPaths["merged"], "_1_.txt"), expectedOverlayNumLower); err != nil {
			t.Fatal(err)
		}
		if err := testOpenWriteFile(test, path.Join(testDriveOverlay.overlayPaths["merged"], "__2.txt"), expectedOverlayNumLower); err != nil {
			t.Fatal(err)
		}
	})
}

func TestOverlayWriteMergedOnly(t *testing.T) {
	// Don't run the test inside Docker
	if testEnvironment == DockerEnvironment {
		t.Skip()

	}

	// Create the Overlay Drive with 3 lower layers
	testDriveOverlay, err := newTestDriveOverlay(3)
	if err != nil {
		t.Fatal(err)
	}
	defer testDriveOverlay.Close()

	// Open files for writing in lower layers
	rule := &rules.RuleDefinition{
		ID:         "overlay_upper_write_rule",
		Expression: `open.filename == "{{.Root}}/0__.txt" || open.filename == "{{.Root}}/0_2.txt" || open.filename == "{{.Root}}/_1_.txt" || open.filename == "{{.Root}}/__2.txt"`,
	}

	test, err := newTestModule(nil, []*rules.RuleDefinition{rule}, testOpts{testDir: testDriveOverlay.overlayPaths["merged"]})
	if err != nil {
		t.Fatal(err)
	}
	defer test.Close()

	t.Run("overlay_write_upper", func(t *testing.T) {
		var expectedOverlayNumLower int32 = 0
		if err := testOpenWriteFile(test, path.Join(testDriveOverlay.overlayPaths["merged"], "0__.txt"), expectedOverlayNumLower); err != nil {
			t.Fatal(err)
		}
		if err := testOpenWriteFile(test, path.Join(testDriveOverlay.overlayPaths["merged"], "0_2.txt"), expectedOverlayNumLower); err != nil {
			t.Fatal(err)
		}
		if err := testOpenWriteFile(test, path.Join(testDriveOverlay.overlayPaths["merged"], "_1_.txt"), expectedOverlayNumLower); err != nil {
			t.Fatal(err)
		}
		if err := testOpenWriteFile(test, path.Join(testDriveOverlay.overlayPaths["merged"], "__2.txt"), expectedOverlayNumLower); err != nil {
			t.Fatal(err)
		}
	})
}

func TestOverlayRemoveWriteFromUpper(t *testing.T) {
	// Don't run the test inside Docker
	if testEnvironment == DockerEnvironment {
		t.Skip()

	}

	// Create the Overlay Drive with one lower layer
	testDriveOverlay, err := newTestDriveOverlay(1)
	if err != nil {
		t.Fatal(err)
	}
	defer testDriveOverlay.Close()

	// Create a file in the lower layer
	pathname := path.Join(testDriveOverlay.lowerLayersPaths[0], "0__.txt")
	fmt.Println(pathname)
	if err := createFile(pathname); err != nil {
		t.Fatal(err)
	}

	// Remove and create a file
	rule_1 := &rules.RuleDefinition{
		ID:         "overlay_remove_rule",
		Expression: `unlink.filename == "{{.Root}}/0__.txt" || unlink.filename == "{{.Root}}/new.txt"`,
	}
	rule_2 := &rules.RuleDefinition{
		ID:         "overlay_create_rule",
		Expression: `open.filename == "{{.Root}}/0__.txt" || open.filename == "{{.Root}}/new.txt"`,
	}

	test, err := newTestModule(nil, []*rules.RuleDefinition{rule_1, rule_2}, testOpts{testDir: testDriveOverlay.overlayPaths["merged"]})
	if err != nil {
		t.Fatal(err)
	}
	defer test.Close()

	t.Run("overlay_remove_create_lower", func(t *testing.T) {
		filename := path.Join(testDriveOverlay.overlayPaths["merged"], "0__.txt")
		if err := testOpenReadFile(test, filename, 1); err != nil {
			t.Fatal(err)
		}

		os.Remove(filename)
		event, _, err := test.GetEvent()
		if err != nil {
			t.Fatal(err)
		}
		fmt.Printf("Event: %v\n", event)
		if event.Unlink.OverlayNumLower != 1 {
			t.Fatal(errors.Errorf("expected OverlayNumLower %d, got %d", 1, event.Unlink.OverlayNumLower))
		}

		createFile(filename)
		event, _, err = test.GetEvent()
		if err != nil {
			t.Fatal(err)
		}
		fmt.Printf("Event: %v\n", event)
		if event.Open.OverlayNumLower != 0 {
			t.Fatal(errors.Errorf("expected OverlayNumLower %d, got %d", 0, event.Open.OverlayNumLower))
		}

		os.Remove(filename)
		event, _, err = test.GetEvent()
		if err != nil {
			t.Fatal(err)
		}
		fmt.Printf("Event: %v\n", event)
		if event.Unlink.OverlayNumLower != 0 {
			t.Fatal(errors.Errorf("expected OverlayNumLower %d, got %d", 0, event.Unlink.OverlayNumLower))
		}

		filename = path.Join(testDriveOverlay.overlayPaths["merged"], "new.txt")
		createFile(filename)
		event, _, err = test.GetEvent()
		if err != nil {
			t.Fatal(err)
		}
		fmt.Printf("Event: %v\n", event)
		if event.Open.OverlayNumLower != 0 {
			t.Fatal(errors.Errorf("expected OverlayNumLower %d, got %d", 0, event.Open.OverlayNumLower))
		}

		os.Remove(filename)
		event, _, err = test.GetEvent()
		if err != nil {
			t.Fatal(err)
		}
		fmt.Printf("Event: %v\n", event)
		if event.Unlink.OverlayNumLower != 0 {
			t.Fatal(errors.Errorf("expected OverlayNumLower %d, got %d", 0, event.Unlink.OverlayNumLower))
		}

	})
}

func TestOverlayOpenWriteOpenFromUpper(t *testing.T) {
	// Don't run the test inside Docker
	if testEnvironment == DockerEnvironment {
		t.Skip()

	}

	// Create the Overlay Drive with one lower layer
	testDriveOverlay, err := newTestDriveOverlay(1)
	if err != nil {
		t.Fatal(err)
	}
	defer testDriveOverlay.Close()

	// Create a file in the lower layer
	pathname := path.Join(testDriveOverlay.lowerLayersPaths[0], "0__.txt")
	fmt.Println(pathname)
	if err := createFile(pathname); err != nil {
		t.Fatal(err)
	}

	// Modify an existing file
	rule := &rules.RuleDefinition{
		ID:         "overlay_modify_rule",
		Expression: `open.filename == "{{.Root}}/0__.txt"`,
	}

	test, err := newTestModule(nil, []*rules.RuleDefinition{rule}, testOpts{testDir: testDriveOverlay.overlayPaths["merged"]})
	if err != nil {
		t.Fatal(err)
	}
	defer test.Close()

	t.Run("overlay_modify_lower", func(t *testing.T) {
		filename := path.Join(testDriveOverlay.overlayPaths["merged"], "0__.txt")
		f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE, 0400)
		if err != nil {
			t.Fatal(err)
		}
		f.WriteString("merged")
		f.Close()
		event, _, err := test.GetEvent()
		if err != nil {
			t.Fatal(err)
		}
		fmt.Printf("Event: %v\n", event)
		if event.Open.OverlayNumLower != 0 {
			t.Fatal(errors.Errorf("expected OverlayNumLower %d, got %d", 0, event.Open.OverlayNumLower))
		}

		if err := testOpenReadFile(test, filename, 0); err != nil {
			t.Fatal(err)
		}
	})
}
