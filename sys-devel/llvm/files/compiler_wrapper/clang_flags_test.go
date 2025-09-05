// Copyright 2019 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"testing"
)

func TestClangBasename(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		var tests = []struct {
			in  string
			out string
		}{
			{"./x86_64-cros-linux-gnu-clang", ".*/clang"},
			{"./x86_64-cros-linux-gnu-clang++", ".*/clang\\+\\+"},
		}

		for _, tt := range tests {
			cmd := ctx.must(callCompiler(ctx, ctx.cfg,
				ctx.newCommand(tt.in, mainCc)))
			if err := verifyPath(cmd, tt.out); err != nil {
				t.Error(err)
			}
		}
	})
}

func TestAbsoluteClangPathBasedOnRootPath(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		ctx.cfg.clangRootRelPath = "somepath"
		cmd := ctx.must(callCompiler(ctx, ctx.cfg,
			ctx.newCommand(filepath.Join(ctx.tempDir, clangX86_64), mainCc)))
		if err := verifyPath(cmd, filepath.Join(ctx.tempDir, "somepath/usr/bin/clang")); err != nil {
			t.Error(err)
		}
	})
}

func TestRelativeClangPathBasedOnRootPath(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		ctx.cfg.clangRootRelPath = "somepath"
		cmd := ctx.must(callCompiler(ctx, ctx.cfg,
			ctx.newCommand(clangX86_64, mainCc)))
		if err := verifyPath(cmd, "somepath/usr/bin/clang"); err != nil {
			t.Error(err)
		}
	})
}

func TestRelativeClangPathWithDirBasedOnRootPath(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		ctx.cfg.clangRootRelPath = "somepath"
		cmd := ctx.must(callCompiler(ctx, ctx.cfg,
			ctx.newCommand("test/x86_64-cros-linux-gnu-clang", mainCc)))
		if err := verifyPath(cmd, "test/somepath/usr/bin/clang"); err != nil {
			t.Error(err)
		}
	})
}

func TestPathEnvClangPathBasedOnRootPath(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		ctx.cfg.clangRootRelPath = "somepath"
		ctx.env = []string{"PATH=" + filepath.Join(ctx.tempDir, "/pathenv")}
		ctx.writeFile(filepath.Join(ctx.tempDir, "/pathenv/x86_64-cros-linux-gnu-clang"), "")
		cmd := ctx.must(callCompiler(ctx, ctx.cfg,
			ctx.newCommand("x86_64-cros-linux-gnu-clang", mainCc)))
		if err := verifyPath(cmd, filepath.Join(ctx.tempDir, "pathenv/somepath/usr/bin/clang")); err != nil {
			t.Error(err)
		}
	})
}

func TestClangPathForClangHostWrapper(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		ctx.cfg.isHostWrapper = true
		ctx.cfg.clangRootRelPath = "somepath"
		cmd := ctx.must(callCompiler(ctx, ctx.cfg,
			ctx.newCommand(clangX86_64, mainCc)))
		if err := verifyPath(cmd, filepath.Join(ctx.tempDir, "clang")); err != nil {
			t.Error(err)
		}
	})
}

func TestClangPathForAndroidWrapper(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		ctx.cfg.isAndroidWrapper = true
		cmd := ctx.must(callCompiler(ctx, ctx.cfg,
			ctx.newCommand("somedir/clang", mainCc)))
		if err := verifyPath(cmd, "somedir/clang-real"); err != nil {
			t.Error(err)
		}
	})
}

func TestClangPathForAndroidWrapperWithSymlinks(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		ctx.cfg.isAndroidWrapper = true
		ctx.writeFile("base/come_clang", "")
		ctx.symlink("base/some_clang", "linked/clang")
		cmd := ctx.must(callCompiler(ctx, ctx.cfg,
			ctx.newCommand("linked/clang", mainCc)))
		if err := verifyPath(cmd, "linked/some_clang-real"); err != nil {
			t.Error(err)
		}
	})
}

func TestUseXclangPathAndCalcResourceDirByNestedClangCall(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		ctx.cfg.clangRootRelPath = "somepath"
		ctx.cmdMock = func(cmd *command, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
			if ctx.cmdCount > 1 {
				return nil
			}
			if err := verifyPath(cmd, "somepath/usr/bin/clang"); err != nil {
				t.Error(err)
			}
			if err := verifyArgOrder(cmd, "--print-resource-dir"); err != nil {
				t.Error(err)
			}
			fmt.Fprint(stdout, "someResourcePath\n")
			return nil
		}
		cmd := ctx.must(callCompiler(ctx, ctx.cfg,
			ctx.newCommand(clangX86_64, "-Xclang-path=somedir", mainCc)))
		if err := verifyPath(cmd, "somedir/clang"); err != nil {
			t.Error(err)
		}
		if err := verifyArgOrder(cmd, "-resource-dir=someResourcePath",
			"--gcc-toolchain=/usr", mainCc); err != nil {
			t.Error(err)
		}
	})
}

func TestXclangPathFailIfNestedClangCallFails(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		ctx.cmdMock = func(cmd *command, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
			fmt.Fprint(stderr, "someclangerror")
			return errors.New("someerror")
		}
		stderr := ctx.mustFail(callCompiler(ctx, ctx.cfg,
			ctx.newCommand(clangX86_64, "-Xclang-path=somedir", mainCc)))
		if err := verifyInternalError(stderr); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(stderr, "clang") {
			t.Errorf("could not find compiler path on stderr. Got: %s", stderr)
		}
		if !strings.Contains(stderr, "someerror") {
			t.Errorf("could not find original error on stderr. Got: %s", stderr)
		}
		if !strings.Contains(stderr, "someclangerror") {
			t.Errorf("stderr was not forwarded. Got: %s", stderr)
		}
	})
}

func TestConvertGccToClangFlags(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		var tests = []struct {
			in  string
			out string
		}{
			{"-Wno-error=maybe-uninitialized", "-Wno-error=uninitialized"},
			{"-Wno-error=cpp", "-Wno-#warnings"},
			{"-Xclang-only=-abc=xyz", "-abc=xyz"},
		}

		for _, tt := range tests {
			cmd := ctx.must(callCompiler(ctx, ctx.cfg,
				ctx.newCommand(clangX86_64, tt.in, mainCc)))
			if err := verifyArgCount(cmd, 0, tt.in); err != nil {
				t.Error(err)
			}
			if err := verifyArgOrder(cmd, tt.out, mainCc); err != nil {
				t.Error(err)
			}
		}
	})
}

func TestFilterUnsupportedClangFlags(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		var tests = []struct {
			compiler      string
			flag          string
			expectedCount int
		}{
			{clangX86_64, "-Wstrict-aliasing=xyz", 0},
			{clangX86_64, "-finline-limit=xyz", 0},
			{"./armv7a-cros-linux-gnu-clang", "-ftrapv", 0},
			{"./armv7a-cros-win-gnu-clang", "-ftrapv", 1},
			{"./armv8a-cros-win-gnu-clang", "-ftrapv", 1},
			{clangX86_64, "-ftrapv", 1},
		}

		for _, tt := range tests {
			cmd := ctx.must(callCompiler(ctx, ctx.cfg,
				ctx.newCommand(tt.compiler, tt.flag, mainCc)))
			if err := verifyArgCount(cmd, tt.expectedCount, tt.flag); err != nil {
				t.Error(err)
			}
		}
	})
}

func TestClangArchFlags(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		var tests = []struct {
			compiler string
			flags    []string
		}{
			{"./i686_64-cros-linux-gnu-clang", []string{mainCc, "-target", "i686_64-cros-linux-gnu"}},
			{"./x86_64-cros-linux-gnu-clang", []string{mainCc, "-target", "x86_64-cros-linux-gnu"}},
		}
		for _, tt := range tests {
			cmd := ctx.must(callCompiler(ctx, ctx.cfg,
				ctx.newCommand(tt.compiler, mainCc)))
			if err := verifyArgOrder(cmd, tt.flags...); err != nil {
				t.Error(err)
			}
		}
	})
}

func TestClangLinkerPathProbesBinariesOnPath(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		linkerPath := filepath.Join(ctx.tempDir, "a/b/c")
		ctx.writeFile(filepath.Join(linkerPath, "x86_64-cros-linux-gnu-ld.bfd"), "")
		ctx.env = []string{"PATH=nonExistantPath:" + linkerPath}
		cmd := ctx.must(callCompiler(ctx, ctx.cfg,
			ctx.newCommand("./x86_64-cros-linux-gnu-clang", mainCc)))
		if err := verifyArgOrder(cmd, "-Ba/b/c"); err != nil {
			t.Error(err)
		}
		if err := verifyArgOrder(cmd, "--prefix=a/b/c/x86_64-cros-linux-gnu-"); err != nil {
			t.Error(err)
		}

	})
}

func TestClangLinkerPathEvaluatesSymlinksForBinariesOnPath(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		realLinkerPath := filepath.Join(ctx.tempDir, "a/original/path/somelinker")
		ctx.writeFile(realLinkerPath, "")
		firstLinkLinkerPath := filepath.Join(ctx.tempDir, "a/first/somelinker")
		ctx.symlink(realLinkerPath, firstLinkLinkerPath)
		secondLinkLinkerPath := filepath.Join(ctx.tempDir, "a/second/x86_64-cros-linux-gnu-ld.bfd")
		ctx.symlink(firstLinkLinkerPath, secondLinkLinkerPath)

		ctx.env = []string{"PATH=nonExistantPath:" + filepath.Dir(secondLinkLinkerPath)}
		cmd := ctx.must(callCompiler(ctx, ctx.cfg,
			ctx.newCommand("./x86_64-cros-linux-gnu-clang", mainCc)))
		if err := verifyArgOrder(cmd, "-Ba/first"); err != nil {
			t.Error(err)
		}
		if err := verifyArgOrder(cmd, "--prefix=a/first/x86_64-cros-linux-gnu-"); err != nil {
			t.Error(err)
		}

	})
}

func TestClangFallbackLinkerPathRelativeToRootDir(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		cmd := ctx.must(callCompiler(ctx, ctx.cfg,
			ctx.newCommand(clangX86_64, mainCc)))
		if err := verifyArgOrder(cmd, "-Bbin"); err != nil {
			t.Error(err)
		}
		if err := verifyArgOrder(cmd, "--prefix=bin/x86_64-cros-linux-gnu-"); err != nil {
			t.Error(err)
		}
	})
}

func TestClangLinkerPathRelativeToRootDir(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		ctx.cfg.clangRootRelPath = "somepath"
		cmd := ctx.must(callCompiler(ctx, ctx.cfg,
			ctx.newCommand(clangX86_64, mainCc)))
		if err := verifyArgOrder(cmd, "-Bsomepath/bin"); err != nil {
			t.Error(err)
		}
		if err := verifyArgOrder(cmd, "--prefix=somepath/bin/x86_64-cros-linux-gnu-"); err != nil {
			t.Error(err)
		}
	})
}

func TestFilterUnsupportedFlags(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		cmd := ctx.must(callCompiler(ctx, ctx.cfg,
			ctx.newCommand(clangX86_64, "-Xcompiler", mainCc)))
		if err := verifyArgCount(cmd, 0, "-Xcompiler"); err != nil {
			t.Error(err)
		}
		cmd = ctx.must(callCompiler(ctx, ctx.cfg,
			ctx.newCommand(clangX86_64, "-avoid-version", mainCc)))
		if err := verifyArgCount(cmd, 0, "-avoid-version"); err != nil {
			t.Error(err)
		}
	})
}
