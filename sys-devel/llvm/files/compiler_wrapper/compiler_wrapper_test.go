// Copyright 2019 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func TestAddCommonFlags(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		ctx.cfg.commonFlags = []string{"-someflag"}
		cmd := ctx.must(callCompiler(ctx, ctx.cfg,
			ctx.newCommand(gccX86_64, mainCc)))
		if err := verifyArgOrder(cmd, "-someflag", mainCc); err != nil {
			t.Error(err)
		}
	})
}

func TestAddGccConfigFlags(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		ctx.cfg.gccFlags = []string{"-someflag"}
		cmd := ctx.must(callCompiler(ctx, ctx.cfg,
			ctx.newCommand(gccX86_64, mainCc)))
		if err := verifyArgOrder(cmd, "-someflag", mainCc); err != nil {
			t.Error(err)
		}
	})
}

func TestAddClangConfigFlags(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		ctx.cfg.clangFlags = []string{"-someflag"}
		cmd := ctx.must(callCompiler(ctx, ctx.cfg,
			ctx.newCommand(clangX86_64, mainCc)))
		if err := verifyArgOrder(cmd, "-someflag", mainCc); err != nil {
			t.Error(err)
		}
	})
}

func TestLogGeneralExecError(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		ctx.cmdMock = func(cmd *command, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
			return errors.New("someerror")
		}
		stderr := ctx.mustFail(callCompiler(ctx, ctx.cfg, ctx.newCommand(gccX86_64, mainCc)))
		if err := verifyInternalError(stderr); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(stderr, gccX86_64) {
			t.Errorf("could not find compiler path on stderr. Got: %s", stderr)
		}
		if !strings.Contains(stderr, "someerror") {
			t.Errorf("could not find original error on stderr. Got: %s", stderr)
		}
	})
}

func TestForwardStdin(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		io.WriteString(&ctx.stdinBuffer, "someinput")
		ctx.cmdMock = func(cmd *command, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
			stdinStr := ctx.readAllString(stdin)
			if stdinStr != "someinput" {
				return fmt.Errorf("unexpected stdin. Got: %s", stdinStr)
			}
			return nil
		}
		ctx.must(callCompiler(ctx, ctx.cfg, ctx.newCommand(gccX86_64, "-", mainCc)))
	})
}

func TestLogMissingCCacheExecError(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		ctx.cfg.useCCache = true

		ctx.cmdMock = func(cmd *command, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
			return syscall.ENOENT
		}
		ctx.stderrBuffer.Reset()
		stderr := ctx.mustFail(callCompiler(ctx, ctx.cfg, ctx.newCommand(gccX86_64, mainCc)))
		if err := verifyNonInternalError(stderr, "ccache not found under .*. Please install it"); err != nil {
			t.Fatal(err)
		}
	})
}

func TestGomaDisablesRusage(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		gomaPath := path.Join(ctx.tempDir, "gomacc")
		ctx.writeFile(gomaPath, "")
		ctx.env = []string{"GOMACC_PATH=" + gomaPath}
		logFileName := filepath.Join(ctx.tempDir, "rusage.log")
		ctx.env = []string{
			"TOOLCHAIN_RUSAGE_OUTPUT=" + logFileName,
			"GOMACC_PATH=" + gomaPath,
		}
		cmd := ctx.must(callCompiler(ctx, ctx.cfg, ctx.newCommand(gccX86_64, mainCc)))
		// Ensure Goma was used
		if err := verifyPath(cmd, gomaPath); err != nil {
			t.Fatal(err)
		}
		if err := verifyArgOrder(cmd, gccX86_64+".real", mainCc); err != nil {
			t.Error(err)
		}
		// Ensure rusage log was not created
		if _, err := os.Stat(logFileName); err == nil {
			t.Errorf("Logfile shouldn't have been created at TOOLCHAIN_RUSAGE_OUTPUT path %q but was", logFileName)
		} else if !os.IsNotExist(err) {
			t.Fatalf("error checking for rusage logfile at %q: %v", logFileName, err)
		}
	})
}

func TestLogRusageAndForceDisableWError(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		ctx.NoteTestWritesToUmask()

		logFileName := filepath.Join(ctx.tempDir, "rusage.log")
		ctx.env = []string{
			"FORCE_DISABLE_WERROR=1",
			"TOOLCHAIN_RUSAGE_OUTPUT=" + logFileName,
		}
		ctx.cmdMock = func(cmd *command, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
			switch ctx.cmdCount {
			case 1:
				io.WriteString(stderr, arbitraryWerrorStderr)
				return newExitCodeError(1)
			case 2:
				return nil
			default:
				t.Fatalf("unexpected command: %#v", cmd)
				return nil
			}
		}
		ctx.must(callCompiler(ctx, ctx.cfg, ctx.newCommand(clangX86_64, mainCc)))
		if _, err := os.Stat(logFileName); os.IsNotExist(err) {
			t.Errorf("no logfile created at TOOLCHAIN_RUSAGE_OUTPUT path %q", logFileName)
		} else if err != nil {
			t.Fatalf("error checking for rusage logfile at %q: %v", logFileName, err)
		}
		if ctx.cmdCount != 2 {
			t.Errorf("expected 2 calls. Got: %d", ctx.cmdCount)
		}
	})
}

func TestPrintUserCompilerError(t *testing.T) {
	buffer := bytes.Buffer{}
	printCompilerError(&buffer, newUserErrorf("abcd"))
	if buffer.String() != "abcd\n" {
		t.Errorf("Unexpected string. Got: %s", buffer.String())
	}
}

func TestPrintOtherCompilerError(t *testing.T) {
	buffer := bytes.Buffer{}
	printCompilerError(&buffer, errors.New("abcd"))
	if buffer.String() != "Internal error. Please report to chromeos-toolchain@google.com.\nabcd\n" {
		t.Errorf("Unexpected string. Got: %s", buffer.String())
	}
}

func TestPrintOtherCompilerErrorForAndroidLLVM(t *testing.T) {
	buffer := bytes.Buffer{}

	oldConfigName := ConfigName
	defer func() { ConfigName = oldConfigName }()

	ConfigName = "android"
	printCompilerError(&buffer, errors.New("abcd"))
	if buffer.String() != "Internal error. Please report to android-llvm@google.com.\nabcd\n" {
		t.Errorf("Unexpected string. Got: %s", buffer.String())
	}
}

func TestCalculateAndroidWrapperPath(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		mainBuilderPath string
		absWrapperPath  string
		want            string
	}{
		{
			mainBuilderPath: "/foo/bar",
			absWrapperPath:  "/bar/baz",
			want:            "/foo/baz-real",
		},
		{
			mainBuilderPath: "/my_wrapper",
			absWrapperPath:  "/bar/baz",
			want:            "/baz-real",
		},
		{
			mainBuilderPath: "no_seps",
			absWrapperPath:  "/bar/baz",
			want:            "baz-real",
		},
		{
			mainBuilderPath: "./a_sep",
			absWrapperPath:  "/bar/baz",
			want:            "./baz-real",
		},
	}

	for _, tc := range testCases {
		if result := calculateAndroidWrapperPath(tc.mainBuilderPath, tc.absWrapperPath); result != tc.want {
			t.Errorf("Failed calculating the wrapper path with (%q, %q); got %q, want %q", tc.mainBuilderPath, tc.absWrapperPath, result, tc.want)
		}
	}
}

// If "crash-diagnostics-dir" flag is not provided, add one in
func TestCrashDiagDefault(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		ctx.env = []string{
			"CROS_ARTIFACTS_TMP_DIR=/tmp/foo",
		}
		cmd := ctx.must(callCompiler(ctx, ctx.cfg,
			ctx.newCommand(clangX86_64, mainCc)))

		// Verify that we added the default flag
		if err := verifyArgCount(cmd, 1, "-fcrash-diagnostics-dir=/tmp/foo/toolchain/clang_crash_diagnostics"); err != nil {
			t.Error(err)
		}
	})
}

// If "crash-diagnostics-dir" flag is already provided by the user, only use that flag and don't add a dupe
func TestCrashDiagUserFlag(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		ctx.env = []string{
			"CROS_ARTIFACTS_TMP_DIR=/tmp/foo",
		}
		cmd := ctx.must(callCompiler(ctx, ctx.cfg,
			ctx.newCommand(clangX86_64, mainCc, "-fcrash-diagnostics-dir=/build/something/foozz")))

		// Verify that user flag is not removed
		if err := verifyArgCount(cmd, 1, "-fcrash-diagnostics-dir=/build/something/foozz"); err != nil {
			t.Error(err)
		}

		// Verify that we did not add the default flag
		if err := verifyArgCount(cmd, 0, "-fcrash-diagnostics-dir=/tmp/foo/toolchain/clang_crash_diagnostics"); err != nil {
			t.Error(err)
		}
	})
}
