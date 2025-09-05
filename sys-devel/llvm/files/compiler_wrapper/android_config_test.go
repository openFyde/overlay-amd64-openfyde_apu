// Copyright 2019 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"path"
	"path/filepath"
	"testing"
)

const androidGoldenDir = "testdata/android_golden"

func TestAndroidConfigDoesNotSpecifyCrashDir(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		useLlvmNext := false
		useCCache := false
		cfg, err := getConfig("android", useCCache, useLlvmNext, "123")
		if err != nil {
			t.Fatal(err)
		}
		ctx.updateConfig(cfg)

		cmd := ctx.must(callCompiler(ctx, ctx.cfg, ctx.newCommand(clangX86_64, mainCc)))
		if err := verifyArgCount(cmd, 0, "-fcrash-diagnostics-dir=.*"); err != nil {
			t.Error(err)
		}
	})
}

func TestAndroidConfig(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		useLlvmNext := false
		useCCache := false
		cfg, err := getConfig("android", useCCache, useLlvmNext, "123")
		if err != nil {
			t.Fatal(err)
		}
		ctx.updateConfig(cfg)

		runGoldenRecords(ctx, androidGoldenDir, []goldenFile{
			createAndroidClangPathGoldenInputs(ctx),
			createAndroidCompileWithFallbackGoldenInputs(ctx),
		})
	})
}

func createAndroidClangPathGoldenInputs(ctx *testContext) goldenFile {
	gomaPath := path.Join(ctx.tempDir, "gomacc")
	ctx.writeFile(gomaPath, "")
	defaultPath := filepath.Join(ctx.tempDir, "clang")
	clangTidyPath := filepath.Join(ctx.tempDir, "clang-tidy")

	deepPath := "a/b/c/d/e/f/g/clang"
	linkedDeepPath := "symlinked/clang_other"
	ctx.writeFile(filepath.Join(ctx.tempDir, "/pathenv/clang"), "")
	ctx.symlink(deepPath, linkedDeepPath)
	return goldenFile{
		Name: "clang_path.json",
		Records: []goldenRecord{
			{
				WrapperCmd: newGoldenCmd(defaultPath, mainCc),
				Cmds:       okResults,
			},
			{
				WrapperCmd: newGoldenCmd(defaultPath, mainCc),
				Cmds:       errorResults,
			},
			{
				Env:        []string{"WITH_TIDY=1"},
				WrapperCmd: newGoldenCmd(defaultPath, mainCc),
				Cmds:       okResults,
			},
			{
				WrapperCmd: newGoldenCmd(filepath.Join(ctx.tempDir, "clang++"), mainCc),
				Cmds:       okResults,
			},
			{
				WrapperCmd: newGoldenCmd(clangTidyPath, mainCc),
				Cmds:       okResults,
			},
			{
				Env:        []string{"WITH_TIDY=1"},
				WrapperCmd: newGoldenCmd(clangTidyPath, mainCc),
				Cmds:       okResults,
			},
			{
				WrapperCmd: newGoldenCmd(deepPath, mainCc),
				Cmds:       okResults,
			},
			{
				WrapperCmd: newGoldenCmd(linkedDeepPath, mainCc),
				Cmds:       okResults,
			},
			{
				Env:        []string{"PATH=" + filepath.Join(ctx.tempDir, "/pathenv")},
				WrapperCmd: newGoldenCmd("clang", mainCc),
				Cmds:       okResults,
			},
			{
				WrapperCmd: newGoldenCmd(defaultPath, mainCc, "--gomacc-path", gomaPath),
				Cmds:       okResults,
			},
		},
	}
}

func createAndroidCompileWithFallbackGoldenInputs(ctx *testContext) goldenFile {
	env := []string{
		"ANDROID_LLVM_PREBUILT_COMPILER_PATH=fallback_compiler",
		"ANDROID_LLVM_STDERR_REDIRECT=" + filepath.Join(ctx.tempDir, "fallback_stderr"),
		"ANDROID_LLVM_FALLBACK_DISABLED_WARNINGS=-a -b",
	}
	defaultPath := filepath.Join(ctx.tempDir, "clang")
	return goldenFile{
		Name: "compile_with_fallback.json",
		Records: []goldenRecord{
			{
				WrapperCmd: newGoldenCmd(defaultPath, mainCc),
				Env:        env,
				Cmds:       okResults,
			},
			{
				WrapperCmd: newGoldenCmd(defaultPath, mainCc),
				Env:        env,
				Cmds: []commandResult{
					{
						ExitCode: 1,
					},
					okResult,
				},
			},
			{
				WrapperCmd: newGoldenCmd(defaultPath, mainCc),
				Env:        env,
				Cmds: []commandResult{
					{
						ExitCode: 1,
					},
					{
						ExitCode: 1,
					},
				},
			},
		},
	}
}
