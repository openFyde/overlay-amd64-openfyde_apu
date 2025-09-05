// Copyright 2019 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

// Save this off before goroutines start running, since this necessarily involves modifying the
// value for our umask, and that screams subtle race conditions. :)
var umaskAtStartup = func() os.FileMode {
	umask := syscall.Umask(0)
	syscall.Umask(umask)
	return os.FileMode(umask)
}()

func TestOmitFallbackCompileForSuccessfulCall(t *testing.T) {
	withCompileWithFallbackTestContext(t, func(ctx *testContext) {
		ctx.must(callCompiler(ctx, ctx.cfg, ctx.newCommand(clangAndroid, mainCc)))
		if ctx.cmdCount != 1 {
			t.Errorf("expected 1 call. Got: %d", ctx.cmdCount)
		}
	})
}

func TestOmitFallbackCompileForGeneralError(t *testing.T) {
	withCompileWithFallbackTestContext(t, func(ctx *testContext) {
		ctx.cmdMock = func(cmd *command, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
			return errors.New("someerror")
		}
		stderr := ctx.mustFail(callCompiler(ctx, ctx.cfg, ctx.newCommand(clangAndroid, mainCc)))
		if err := verifyInternalError(stderr); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(stderr, "someerror") {
			t.Errorf("unexpected error. Got: %s", stderr)
		}
		if ctx.cmdCount != 1 {
			t.Errorf("expected 1 call. Got: %d", ctx.cmdCount)
		}
	})
}

func TestCompileWithFallbackForNonZeroExitCode(t *testing.T) {
	withCompileWithFallbackTestContext(t, func(ctx *testContext) {
		ctx.cmdMock = func(cmd *command, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
			switch ctx.cmdCount {
			case 1:
				return newExitCodeError(1)
			case 2:
				if err := verifyPath(cmd, "fallback_compiler/clang"); err != nil {
					return err
				}
				if err := verifyEnvUpdate(cmd, "ANDROID_LLVM_PREBUILT_COMPILER_PATH="); err != nil {
					return err
				}
				return nil
			default:
				t.Fatalf("unexpected command: %#v", cmd)
				return nil
			}
		}
		ctx.must(callCompiler(ctx, ctx.cfg, ctx.newCommand(clangAndroid, mainCc)))
		if ctx.cmdCount != 2 {
			t.Errorf("expected 2 calls. Got: %d", ctx.cmdCount)
		}
	})
}

func TestCompileWithFallbackForwardStdoutAndStderr(t *testing.T) {
	withCompileWithFallbackTestContext(t, func(ctx *testContext) {
		ctx.cmdMock = func(cmd *command, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
			switch ctx.cmdCount {
			case 1:
				fmt.Fprint(stdout, "originalmessage")
				fmt.Fprint(stderr, "originalerror")
				return newExitCodeError(1)
			case 2:
				fmt.Fprint(stdout, "fallbackmessage")
				fmt.Fprint(stderr, "fallbackerror")
				return nil
			default:
				t.Fatalf("unexpected command: %#v", cmd)
				return nil
			}
		}
		ctx.must(callCompiler(ctx, ctx.cfg, ctx.newCommand(clangAndroid, mainCc)))
		if err := verifyNonInternalError(ctx.stderrString(), "originalerrorfallbackerror"); err != nil {
			t.Error(err)
		}
		if !strings.Contains(ctx.stdoutString(), "originalmessagefallbackmessage") {
			t.Errorf("unexpected stdout. Got: %s", ctx.stdoutString())
		}
	})
}

func TestForwardGeneralErrorWhenFallbackCompileFails(t *testing.T) {
	withCompileWithFallbackTestContext(t, func(ctx *testContext) {
		ctx.cmdMock = func(cmd *command, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
			switch ctx.cmdCount {
			case 1:
				return newExitCodeError(1)
			case 2:
				return errors.New("someerror")
			default:
				t.Fatalf("unexpected command: %#v", cmd)
				return nil
			}
		}
		stderr := ctx.mustFail(callCompiler(ctx, ctx.cfg, ctx.newCommand(clangAndroid, mainCc)))
		if err := verifyInternalError(stderr); err != nil {
			t.Error(err)
		}
		if !strings.Contains(stderr, "someerror") {
			t.Errorf("unexpected stderr. Got: %s", stderr)
		}
	})
}

func TestForwardExitCodeWhenFallbackCompileFails(t *testing.T) {
	withCompileWithFallbackTestContext(t, func(ctx *testContext) {
		ctx.cmdMock = func(cmd *command, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
			switch ctx.cmdCount {
			case 1:
				return newExitCodeError(1)
			case 2:
				return newExitCodeError(2)
			default:
				t.Fatalf("unexpected command: %#v", cmd)
				return nil
			}
		}
		exitCode := callCompiler(ctx, ctx.cfg, ctx.newCommand(clangAndroid, mainCc))
		if exitCode != 2 {
			t.Errorf("unexpected exit code. Got: %d", exitCode)
		}
	})
}

func TestForwardStdinToFallbackCompile(t *testing.T) {
	withCompileWithFallbackTestContext(t, func(ctx *testContext) {
		ctx.cmdMock = func(cmd *command, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
			stdinStr := ctx.readAllString(stdin)
			if stdinStr != "someinput" {
				return fmt.Errorf("unexpected stdin. Got: %s", stdinStr)
			}

			switch ctx.cmdCount {
			case 1:
				return newExitCodeError(1)
			case 2:
				return nil
			default:
				t.Fatalf("unexpected command: %#v", cmd)
				return nil
			}
		}
		io.WriteString(&ctx.stdinBuffer, "someinput")
		ctx.must(callCompiler(ctx, ctx.cfg, ctx.newCommand(clangAndroid, "-", mainCc)))
	})
}

func TestCompileWithFallbackExtraArgs(t *testing.T) {
	withCompileWithFallbackTestContext(t, func(ctx *testContext) {
		testData := []struct {
			compiler        string
			expectExtraArgs bool
		}{
			{"./clang", true},
			{"./clang++", true},
			{"./clang-tidy", false},
		}
		ctx.env = append(ctx.env, "ANDROID_LLVM_FALLBACK_DISABLED_WARNINGS=-a -b")
		extraArgs := []string{"-fno-color-diagnostics", "-a", "-b"}
		for _, tt := range testData {
			ctx.cmdCount = 0
			ctx.cmdMock = func(cmd *command, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
				switch ctx.cmdCount {
				case 1:
					if tt.expectExtraArgs {
						if err := verifyArgOrder(cmd, extraArgs...); err != nil {
							return err
						}
					} else {
						for _, arg := range extraArgs {
							if err := verifyArgCount(cmd, 0, arg); err != nil {
								return err
							}
						}
					}
					return newExitCodeError(1)
				case 2:
					for _, arg := range extraArgs {
						if err := verifyArgCount(cmd, 0, arg); err != nil {
							return err
						}
					}
					return nil
				default:
					t.Fatalf("unexpected command: %#v", cmd)
					return nil
				}
			}
			ctx.must(callCompiler(ctx, ctx.cfg, ctx.newCommand(tt.compiler, mainCc)))
			if ctx.cmdCount != 2 {
				t.Errorf("expected 2 calls. Got: %d", ctx.cmdCount)
			}
		}
	})
}

func TestCompileWithFallbackLogCommandAndErrors(t *testing.T) {
	withCompileWithFallbackTestContext(t, func(ctx *testContext) {
		ctx.NoteTestReadsFromUmask()

		ctx.env = append(ctx.env, "ANDROID_LLVM_FALLBACK_DISABLED_WARNINGS=-a -b")
		ctx.cmdMock = func(cmd *command, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
			switch ctx.cmdCount {
			case 1:
				fmt.Fprint(stderr, "someerror\n")
				return newExitCodeError(1)
			case 2:
				return nil
			default:
				t.Fatalf("unexpected command: %#v", cmd)
				return nil
			}
		}
		ctx.must(callCompiler(ctx, ctx.cfg, ctx.newCommand(clangAndroid, mainCc)))

		log := readCompileWithFallbackErrorLog(ctx)
		if log != `==================COMMAND:====================
./clang-real main.cc -fno-color-diagnostics -a -b

someerror
==============================================

` {
			t.Errorf("unexpected log. Got: %s", log)
		}

		entry, _ := os.Lstat(filepath.Join(ctx.tempDir, "fallback_stderr"))
		if entry.Mode()&0777 != 0644 & ^umaskAtStartup {
			t.Errorf("unexpected mode for logfile. Got: %#o", entry.Mode())
		}
	})
}

func TestCompileWithFallbackAppendToLog(t *testing.T) {
	withCompileWithFallbackTestContext(t, func(ctx *testContext) {
		ctx.writeFile(filepath.Join(ctx.tempDir, "fallback_stderr"), "oldContent\n")
		ctx.cmdMock = func(cmd *command, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
			switch ctx.cmdCount {
			case 1:
				return newExitCodeError(1)
			case 2:
				return nil
			default:
				t.Fatalf("unexpected command: %#v", cmd)
				return nil
			}
		}
		ctx.must(callCompiler(ctx, ctx.cfg, ctx.newCommand(clangAndroid, mainCc)))

		log := readCompileWithFallbackErrorLog(ctx)
		if !strings.Contains(log, "oldContent") {
			t.Errorf("old content not present: %s", log)
		}
		if !strings.Contains(log, "clang-real") {
			t.Errorf("new content not present: %s", log)
		}
	})
}

func withCompileWithFallbackTestContext(t *testing.T, work func(ctx *testContext)) {
	withTestContext(t, func(ctx *testContext) {
		ctx.cfg.isAndroidWrapper = true
		ctx.env = []string{
			"ANDROID_LLVM_PREBUILT_COMPILER_PATH=fallback_compiler",
			"ANDROID_LLVM_STDERR_REDIRECT=" + filepath.Join(ctx.tempDir, "fallback_stderr"),
		}
		work(ctx)
	})
}

func readCompileWithFallbackErrorLog(ctx *testContext) string {
	logFile := filepath.Join(ctx.tempDir, "fallback_stderr")
	data, err := os.ReadFile(logFile)
	if err != nil {
		ctx.t.Fatalf("error reading log file %s: %s", logFile, err)
	}
	return string(data)
}
