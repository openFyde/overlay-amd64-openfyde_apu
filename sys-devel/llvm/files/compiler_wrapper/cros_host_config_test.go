//  Copyright 2019 The ChromiumOS Authors
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.

package main

import (
	"path"
	"testing"
)

const crosClangHostGoldenDir = "testdata/cros_clang_host_golden"
const crosGccHostGoldenDir = "testdata/cros_gcc_host_golden"

func TestCrosClangHostConfig(t *testing.T) {
	withGoldenTestContext(t, func(ctx *testContext) {
		useLlvmNext := false
		useCCache := false
		cfg, err := getConfig("cros.host", useCCache, useLlvmNext, "123")
		if err != nil {
			t.Fatal(err)
		}
		ctx.updateConfig(cfg)

		gomaPath := path.Join(ctx.tempDir, "gomacc")
		ctx.writeFile(gomaPath, "")
		gomaEnv := "GOMACC_PATH=" + gomaPath

		goldenFiles := []goldenFile{
			createClangPathGoldenInputs(ctx, gomaEnv),
			createGoldenInputsForAllTargets("clang", mainCc),
			createGoldenInputsForAllTargets("clang", "-ftrapv", mainCc),
			createSanitizerGoldenInputs("clang"),
			createClangArgsGoldenInputs(),
			createForceDisableWErrorGoldenInputs(),
			createClangTidyGoldenInputs(gomaEnv),
			createClangHostWrapperInputs(),
		}

		runGoldenRecords(ctx, crosClangHostGoldenDir, goldenFiles)
	})
}

func TestCrosGccHostConfig(t *testing.T) {
	withGoldenTestContext(t, func(ctx *testContext) {
		useLlvmNext := false
		useCCache := false
		cfg, err := getConfig("cros.host", useCCache, useLlvmNext, "123")
		if err != nil {
			t.Fatal(err)
		}
		ctx.updateConfig(cfg)

		gomaPath := path.Join(ctx.tempDir, "gomacc")
		ctx.writeFile(gomaPath, "")
		gomaEnv := "GOMACC_PATH=" + gomaPath

		// Note: The old gcc host wrapper is very limited and only adds flags.
		// So we only test very few things here.
		goldenFiles := []goldenFile{
			createGccPathGoldenInputs(ctx, gomaEnv),
			createGoldenInputsForAllTargets("gcc", mainCc),
			createGccArgsGoldenInputs(),
			createGccHostWrapperInputs(),
		}

		runGoldenRecords(ctx, crosGccHostGoldenDir, goldenFiles)
	})
}

func createClangHostWrapperInputs() goldenFile {
	return goldenFile{
		Name: "clang_host_wrapper.json",
		Records: []goldenRecord{
			{
				WrapperCmd: newGoldenCmd("./clang", mainCc),
				Cmds:       okResults,
			},
		},
	}
}

func createGccHostWrapperInputs() goldenFile {
	return goldenFile{
		Name: "gcc_host_wrapper.json",
		Records: []goldenRecord{
			{
				WrapperCmd: newGoldenCmd("./gcc", mainCc),
				Cmds:       okResults,
			},
		},
	}
}
