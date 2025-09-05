// Copyright 2019 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"
	"testing"
)

const crosHardenedGoldenDir = "testdata/cros_hardened_golden"
const crosHardenedNoCCacheGoldenDir = "testdata/cros_hardened_noccache_golden"
const crosHardenedLlvmNextGoldenDir = "testdata/cros_hardened_llvmnext_golden"

func withGoldenTestContext(t *testing.T, f func(ctx *testContext)) {
	withTestContext(t, func(ctx *testContext) {
		ctx.NoteTestWritesToUmask()
		f(ctx)
	})
}

func TestCrosHardenedConfig(t *testing.T) {
	withGoldenTestContext(t, func(ctx *testContext) {
		useLlvmNext := false
		useCCache := true
		cfg, err := getConfig("cros.hardened", useCCache, useLlvmNext, "123")
		if err != nil {
			t.Fatal(err)
		}
		ctx.updateConfig(cfg)

		runGoldenRecords(ctx, crosHardenedGoldenDir, createSyswrapperGoldenInputs(ctx))
	})
}

func TestCrosHardenedConfigWithoutCCache(t *testing.T) {
	withGoldenTestContext(t, func(ctx *testContext) {
		useLlvmNext := false
		useCCache := false
		cfg, err := getConfig("cros.hardened", useCCache, useLlvmNext, "123")
		if err != nil {
			t.Fatal(err)
		}
		ctx.updateConfig(cfg)

		// Only run the subset of the sysroot wrapper tests that execute commands.
		gomaPath := path.Join(ctx.tempDir, "gomacc")
		ctx.writeFile(gomaPath, "")
		gomaEnv := "GOMACC_PATH=" + gomaPath
		runGoldenRecords(ctx, crosHardenedNoCCacheGoldenDir, []goldenFile{
			createGccPathGoldenInputs(ctx, gomaEnv),
			createClangPathGoldenInputs(ctx, gomaEnv),
			createClangSyntaxGoldenInputs(gomaEnv),
			createForceDisableWErrorGoldenInputs(),
			createClangTidyGoldenInputs(gomaEnv),
		})
	})
}

func TestCrosHardenedConfigWithLlvmNext(t *testing.T) {
	withGoldenTestContext(t, func(ctx *testContext) {
		useLlvmNext := true
		useCCache := true
		cfg, err := getConfig("cros.hardened", useCCache, useLlvmNext, "123")
		if err != nil {
			t.Fatal(err)
		}
		ctx.updateConfig(cfg)

		// Only run the subset of the sysroot wrapper tests that execute commands.
		gomaPath := path.Join(ctx.tempDir, "gomacc")
		ctx.writeFile(gomaPath, "")
		gomaEnv := "GOMACC_PATH=" + gomaPath
		runGoldenRecords(ctx, crosHardenedLlvmNextGoldenDir, []goldenFile{
			createGccPathGoldenInputs(ctx, gomaEnv),
			createClangPathGoldenInputs(ctx, gomaEnv),
			createClangSyntaxGoldenInputs(gomaEnv),
			createForceDisableWErrorGoldenInputs(),
			createClangTidyGoldenInputs(gomaEnv),
		})
	})
}

func createSyswrapperGoldenInputs(ctx *testContext) []goldenFile {
	gomaPath := path.Join(ctx.tempDir, "gomacc")
	ctx.writeFile(gomaPath, "")
	gomaEnv := "GOMACC_PATH=" + gomaPath

	return []goldenFile{
		createGccPathGoldenInputs(ctx, gomaEnv),
		createGoldenInputsForAllTargets("gcc", mainCc),
		createSysrootWrapperCommonGoldenInputs("gcc", gomaEnv),
		createSanitizerGoldenInputs("gcc"),
		createGccArgsGoldenInputs(),
		createClangSyntaxGoldenInputs(gomaEnv),
		createClangPathGoldenInputs(ctx, gomaEnv),
		createGoldenInputsForAllTargets("clang", mainCc),
		createGoldenInputsForAllTargets("clang", "-ftrapv", mainCc),
		createSysrootWrapperCommonGoldenInputs("clang", gomaEnv),
		createSanitizerGoldenInputs("clang"),
		createClangArgsGoldenInputs(),
		createForceDisableWErrorGoldenInputs(),
		createClangTidyGoldenInputs(gomaEnv),
	}
}

func createGoldenInputsForAllTargets(compiler string, args ...string) goldenFile {
	argsReplacer := strings.NewReplacer(".", "", "-", "")
	return goldenFile{
		Name: fmt.Sprintf("%s_%s_target_specific.json", compiler, argsReplacer.Replace(strings.Join(args, "_"))),
		Records: []goldenRecord{
			{
				WrapperCmd: newGoldenCmd("./x86_64-cros-linux-gnu-"+compiler, args...),
				Cmds:       okResults,
			},
			{
				WrapperCmd: newGoldenCmd("./x86_64-cros-eabi-"+compiler, args...),
				Cmds:       okResults,
			},
			{
				WrapperCmd: newGoldenCmd("./x86_64-cros-win-gnu-"+compiler, args...),
				Cmds:       okResults,
			},
			{
				WrapperCmd: newGoldenCmd("./armv7m-cros-linux-gnu-"+compiler, args...),
				Cmds:       okResults,
			},
			{
				WrapperCmd: newGoldenCmd("./armv7m-cros-eabi-"+compiler, args...),
				Cmds:       okResults,
			},
			{
				WrapperCmd: newGoldenCmd("./armv7m-cros-win-gnu-"+compiler, args...),
				Cmds:       okResults,
			},
			{
				WrapperCmd: newGoldenCmd("./armv8m-cros-linux-gnu-"+compiler, args...),
				Cmds:       okResults,
			},
			{
				WrapperCmd: newGoldenCmd("./armv8m-cros-eabi-"+compiler, args...),
				Cmds:       okResults,
			},
			{
				WrapperCmd: newGoldenCmd("./armv8m-cros-win-gnu-"+compiler, args...),
				Cmds:       okResults,
			},
		},
	}
}

func createForceDisableWErrorGoldenInputs() goldenFile {
	return goldenFile{
		Name: "force_disable_werror.json",
		Records: []goldenRecord{
			{
				WrapperCmd: newGoldenCmd(clangX86_64, mainCc),
				Env:        []string{"FORCE_DISABLE_WERROR=1"},
				Cmds:       okResults,
			},
			{
				WrapperCmd: newGoldenCmd(clangX86_64, mainCc),
				Env:        []string{"FORCE_DISABLE_WERROR=1"},
				Cmds: []commandResult{
					{
						Stderr:   arbitraryWerrorStderr,
						ExitCode: 1,
					},
					okResult,
				},
			},
			{
				WrapperCmd: newGoldenCmd(clangX86_64, mainCc),
				Env:        []string{"FORCE_DISABLE_WERROR=1"},
				Cmds: []commandResult{
					{
						Stderr:   arbitraryWerrorStderr,
						ExitCode: 1,
					},
					errorResult,
				},
			},
		},
	}
}

func createGccPathGoldenInputs(ctx *testContext, gomaEnv string) goldenFile {
	deepPath := "./a/b/c/d/e/f/g/x86_64-cros-linux-gnu-gcc"
	linkedDeepPath := "./symlinked/x86_64-cros-linux-gnu-gcc"
	ctx.writeFile(filepath.Join(ctx.tempDir, "/pathenv/x86_64-cros-linux-gnu-gcc"), "")
	ctx.symlink(deepPath, linkedDeepPath)
	return goldenFile{
		Name: "gcc_path.json",
		Records: []goldenRecord{
			{
				WrapperCmd: newGoldenCmd("./x86_64-cros-linux-gnu-gcc", mainCc),
				Cmds:       okResults,
			},
			{
				WrapperCmd: newGoldenCmd("./x86_64-cros-linux-gnu-gcc", mainCc),
				Cmds:       errorResults,
			},
			{
				WrapperCmd: newGoldenCmd(filepath.Join(ctx.tempDir, "x86_64-cros-linux-gnu-gcc"), mainCc),
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
				WrapperCmd: newGoldenCmd("x86_64-cros-linux-gnu-gcc", mainCc),
				Cmds:       okResults,
			},
		},
	}
}

func createClangPathGoldenInputs(ctx *testContext, gomaEnv string) goldenFile {
	deepPath := "./a/b/c/d/e/f/g/x86_64-cros-linux-gnu-clang"
	linkedDeepPath := "./symlinked/x86_64-cros-linux-gnu-clang"
	ctx.writeFile(filepath.Join(ctx.tempDir, "/pathenv/x86_64-cros-linux-gnu-clang"), "")
	ctx.symlink(deepPath, linkedDeepPath)
	return goldenFile{
		Name: "clang_path.json",
		Records: []goldenRecord{
			{
				WrapperCmd: newGoldenCmd("./x86_64-cros-linux-gnu-clang", mainCc),
				Cmds:       okResults,
			},
			{
				WrapperCmd: newGoldenCmd("./x86_64-cros-linux-gnu-clang", mainCc),
				Cmds:       errorResults,
			},
			{
				WrapperCmd: newGoldenCmd("./x86_64-cros-linux-gnu-clang++", mainCc),
				Cmds:       okResults,
			},
			{
				WrapperCmd: newGoldenCmd(clangX86_64, "-Xclang-path=/somedir", mainCc),
				Cmds: []commandResult{
					{Stdout: "someResourceDir"},
					okResult,
				},
			},
			{
				WrapperCmd: newGoldenCmd(clangX86_64, "-Xclang-path=/somedir", mainCc),
				Env:        []string{gomaEnv},
				Cmds: []commandResult{
					{Stdout: "someResourceDir"},
					okResult,
				},
			},
			{
				WrapperCmd: newGoldenCmd(clangX86_64, "-Xclang-path=/somedir", mainCc),
				Cmds: []commandResult{
					{Stdout: "someResourceDir"},
					errorResult,
				},
			},
			{
				WrapperCmd: newGoldenCmd(filepath.Join(ctx.tempDir, "x86_64-cros-linux-gnu-clang"), mainCc),
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
				WrapperCmd: newGoldenCmd("somedir/x86_64-cros-linux-gnu-clang", mainCc),
				Cmds:       okResults,
			},
			{
				Env:        []string{"PATH=" + filepath.Join(ctx.tempDir, "/pathenv")},
				WrapperCmd: newGoldenCmd("x86_64-cros-linux-gnu-clang", mainCc),
				Cmds:       okResults,
			},
		},
	}
}

func createClangTidyGoldenInputs(gomaEnv string) goldenFile {
	tidyEnv := "WITH_TIDY=1"
	return goldenFile{
		Name: "clangtidy.json",
		Records: []goldenRecord{
			{
				WrapperCmd: newGoldenCmd(clangX86_64, mainCc),
				Env:        []string{tidyEnv},
				Cmds: []commandResult{
					{Stdout: "someResourceDir"},
					okResult,
					okResult,
				},
			},
			{
				WrapperCmd: newGoldenCmd(clangX86_64, mainCc),
				Env:        []string{tidyEnv, gomaEnv},
				Cmds: []commandResult{
					{Stdout: "someResourceDir"},
					okResult,
					okResult,
				},
			},
			{
				WrapperCmd: newGoldenCmd(clangX86_64, mainCc),
				Env:        []string{tidyEnv, gomaEnv},
				Cmds: []commandResult{
					{Stdout: "someResourceDir"},
					errorResult,
					// TODO: we don't fail the compilation due to clang-tidy, as that
					// is the behavior in the old wrapper. Consider changing this in
					// the future.
					okResult,
				},
			},
			{
				WrapperCmd: newGoldenCmd(clangX86_64, mainCc),
				Env:        []string{tidyEnv, gomaEnv},
				Cmds: []commandResult{
					{Stdout: "someResourceDir"},
					okResult,
					errorResult,
				},
			},
		},
	}
}

func createClangSyntaxGoldenInputs(gomaEnv string) goldenFile {
	return goldenFile{
		Name: "gcc_clang_syntax.json",
		Records: []goldenRecord{
			{
				WrapperCmd: newGoldenCmd(gccX86_64, "-clang-syntax", mainCc),
				Cmds: []commandResult{
					okResult,
					okResult,
				},
			},
			{
				WrapperCmd: newGoldenCmd(gccX86_64, "-clang-syntax", mainCc),
				Env:        []string{gomaEnv},
				Cmds: []commandResult{
					okResult,
					okResult,
				},
			},
			{
				WrapperCmd: newGoldenCmd(gccX86_64, "-clang-syntax", mainCc),
				Cmds:       errorResults,
			},
			{
				WrapperCmd: newGoldenCmd(gccX86_64, "-clang-syntax", mainCc),
				Cmds: []commandResult{
					okResult,
					errorResult,
				},
			},
		},
	}
}

func createSysrootWrapperCommonGoldenInputs(compiler string, gomaEnv string) goldenFile {
	// We are using a fixed target as all of the following args are target independent.
	wrapperPath := "./x86_64-cros-linux-gnu-" + compiler
	return goldenFile{
		Name: compiler + "_sysroot_wrapper_common.json",
		Records: []goldenRecord{
			{
				WrapperCmd: newGoldenCmd(gccX86_64, "-noccache", mainCc),
				Cmds:       okResults,
			},
			{
				WrapperCmd: newGoldenCmd(wrapperPath, mainCc),
				Env:        []string{"GOMACC_PATH=someNonExistingPath"},
				Cmds:       okResults,
			},
			{
				WrapperCmd: newGoldenCmd(wrapperPath, mainCc),
				Env:        []string{gomaEnv},
				Cmds:       okResults,
			},
			{
				WrapperCmd: newGoldenCmd(wrapperPath, "-nopie", mainCc),
				Cmds:       okResults,
			},
			{
				WrapperCmd: newGoldenCmd(wrapperPath, "-D__KERNEL__", mainCc),
				Cmds:       okResults,
			},
			{
				WrapperCmd: newGoldenCmd("./armv7a-cros-linux-gnueabihf-"+compiler,
					"-D__KERNEL__", mainCc),
				Cmds: okResults,
			},
			{
				WrapperCmd: newGoldenCmd(wrapperPath, "--sysroot=xyz", mainCc),
				Cmds:       okResults,
			},
		},
	}
}

func createSanitizerGoldenInputs(compiler string) goldenFile {
	// We are using a fixed target as all of the following args are target independent.
	wrapperPath := "./x86_64-cros-linux-gnu-" + compiler
	return goldenFile{
		Name: compiler + "_sanitizer_args.json",
		Records: []goldenRecord{
			{
				WrapperCmd: newGoldenCmd(wrapperPath, "-fsanitize=kernel-address", "-Wl,--no-undefined", mainCc),
				Cmds:       okResults,
			},
			{
				WrapperCmd: newGoldenCmd(wrapperPath, "-fsanitize=kernel-address", "-Wl,-z,defs", mainCc),
				Cmds:       okResults,
			},
			{
				WrapperCmd: newGoldenCmd(wrapperPath, "-fsanitize=kernel-address", "-D_FORTIFY_SOURCE=1", mainCc),
				Cmds:       okResults,
			},
			{
				WrapperCmd: newGoldenCmd(wrapperPath, "-fsanitize=kernel-address", "-D_FORTIFY_SOURCE=2", mainCc),
				Cmds:       okResults,
			},
			{
				WrapperCmd: newGoldenCmd(wrapperPath, "-fsanitize=fuzzer", mainCc),
				Cmds:       okResults,
			},
			{
				WrapperCmd: newGoldenCmd(wrapperPath, "-fsanitize=address", "-fprofile-instr-generate", mainCc),
				Cmds:       okResults,
			},
			{
				WrapperCmd: newGoldenCmd(wrapperPath, "-fsanitize=address", mainCc),
				Cmds:       okResults,
			},
			{
				WrapperCmd: newGoldenCmd(wrapperPath, "-fprofile-instr-generate", mainCc),
				Cmds:       okResults,
			},
		},
	}
}

func createGccArgsGoldenInputs() goldenFile {
	return goldenFile{
		Name: "gcc_specific_args.json",
		Records: []goldenRecord{
			{
				WrapperCmd: newGoldenCmd(gccX86_64, "-march=goldmont", mainCc),
				Cmds:       okResults,
			},
			{
				WrapperCmd: newGoldenCmd(gccX86_64, "-march=goldmont-plus", mainCc),
				Cmds:       okResults,
			},
			{
				WrapperCmd: newGoldenCmd(gccX86_64, "-march=skylake", mainCc),
				Cmds:       okResults,
			},
		},
	}
}

func createClangArgsGoldenInputs() goldenFile {
	return goldenFile{
		Name: "clang_specific_args.json",
		Records: []goldenRecord{
			{
				WrapperCmd: newGoldenCmd(clangX86_64, "-mno-movbe", "-Wclobbered", "-Wno-psabi", "-Wlogical-op",
					"-Wmissing-parameter-type", "-Wold-style-declaration", "-Woverride-init", "-Wunsafe-loop-optimizations",
					"-Wstrict-aliasing=abc", "-finline-limit=abc", mainCc),
				Cmds: okResults,
			},
			{
				WrapperCmd: newGoldenCmd(clangX86_64, "-Wno-error=cpp", mainCc),
				Cmds:       okResults,
			},
			{
				WrapperCmd: newGoldenCmd(clangX86_64, "-Wno-error=maybe-uninitialized", mainCc),
				Cmds:       okResults,
			},
			{
				WrapperCmd: newGoldenCmd(clangX86_64, "-Xclang-only=-someflag", mainCc),
				Cmds:       okResults,
			},
		},
	}
}
