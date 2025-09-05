// Copyright 2019 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestForwardStdOutAndStdErrAndExitCodeFromLogRusage(t *testing.T) {
	withLogRusageTestContext(t, func(ctx *testContext) {
		ctx.cmdMock = func(cmd *command, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
			fmt.Fprint(stdout, "somemessage")
			fmt.Fprint(stderr, "someerror")
			return newExitCodeError(23)
		}
		exitCode := callCompiler(ctx, ctx.cfg, ctx.newCommand(gccX86_64, mainCc))
		if exitCode != 23 {
			t.Errorf("unexpected exit code. Got: %d", exitCode)
		}
		if ctx.stdoutString() != "somemessage" {
			t.Errorf("stdout was not forwarded. Got: %s", ctx.stdoutString())
		}
		if ctx.stderrString() != "someerror" {
			t.Errorf("stderr was not forwarded. Got: %s", ctx.stderrString())
		}
	})
}

func TestForwardStdinFromLogRusage(t *testing.T) {
	withLogRusageTestContext(t, func(ctx *testContext) {
		ctx.cmdMock = func(cmd *command, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
			// Note: This is called for the clang syntax call as well as for
			// the gcc call, and we assert that stdin is cloned and forwarded
			// to both.
			stdinStr := ctx.readAllString(stdin)
			if stdinStr != "someinput" {
				return fmt.Errorf("unexpected stdin. Got: %s", stdinStr)
			}
			return nil
		}
		io.WriteString(&ctx.stdinBuffer, "someinput")
		ctx.must(callCompiler(ctx, ctx.cfg, ctx.newCommand(clangX86_64, "-", mainCc)))
	})
}

func TestReportGeneralErrorsFromLogRusage(t *testing.T) {
	withLogRusageTestContext(t, func(ctx *testContext) {
		ctx.cmdMock = func(cmd *command, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
			return errors.New("someerror")
		}
		stderr := ctx.mustFail(callCompiler(ctx, ctx.cfg,
			ctx.newCommand(gccX86_64, mainCc)))
		if err := verifyInternalError(stderr); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(stderr, "someerror") {
			t.Errorf("unexpected error. Got: %s", stderr)
		}
	})
}

func TestCreateDirAndFileForLogRusage(t *testing.T) {
	withLogRusageTestContext(t, func(ctx *testContext) {
		logFileName := filepath.Join(ctx.tempDir, "somedir", "rusage.log")
		ctx.env = []string{"TOOLCHAIN_RUSAGE_OUTPUT=" + logFileName}
		ctx.must(callCompiler(ctx, ctx.cfg, ctx.newCommand(gccX86_64, mainCc)))

		if _, err := os.Stat(logFileName); err != nil {
			t.Errorf("rusage log file does not exist: %s", err)
		}
	})
}

func TestLogRusageFileContent(t *testing.T) {
	withLogRusageTestContext(t, func(ctx *testContext) {
		logFileName := filepath.Join(ctx.tempDir, "rusage.log")
		ctx.env = []string{"TOOLCHAIN_RUSAGE_OUTPUT=" + logFileName}
		ctx.must(callCompiler(ctx, ctx.cfg, ctx.newCommand(gccX86_64, mainCc)))

		data, err := os.ReadFile(logFileName)
		if err != nil {
			t.Errorf("could not read the rusage log file. Error: %s", err)
		}

		rlog := rusageLog{}

		if err := json.Unmarshal(data, &rlog); err != nil {
			t.Fatalf("rusage log could not be unmarshalled. Got: %s", data)
		}

		if rlog.Compiler != filepath.Join(ctx.tempDir, gccX86_64+".real") {
			t.Errorf("unexpected compiler path. Got: %s", rlog.Compiler)
		}
		if matched, _ := regexp.MatchString("--sysroot=.*", rlog.CompilerArgs[0]); !matched {
			t.Errorf("unexpected compiler args. Got: %s", rlog.CompilerArgs)
		}
		cwd, err := os.Getwd()
		if err != nil {
			t.Fatalf("Failed to get current working directory: %v", err)
		}
		if rlog.WorkingDirectory != cwd {
			t.Errorf("Unexpected working directory. Got: %q, Want: %q", rlog.WorkingDirectory, cwd)
		}
	})
}

func TestLogRusageAppendsToFile(t *testing.T) {
	withLogRusageTestContext(t, func(ctx *testContext) {
		logFileName := filepath.Join(ctx.tempDir, "rusage.log")
		ctx.env = []string{"TOOLCHAIN_RUSAGE_OUTPUT=" + logFileName}

		ctx.must(callCompiler(ctx, ctx.cfg, ctx.newCommand(gccX86_64, mainCc)))
		data, err := os.ReadFile(logFileName)
		if err != nil {
			t.Errorf("could not read the rusage log file. Error: %s", err)
		}
		firstCallLines := strings.Split(string(data), "\n")
		if len(firstCallLines) != 2 {
			t.Errorf("unexpected number of lines. Got: %s", firstCallLines)
		}
		if firstCallLines[0] == "" {
			t.Error("first line was empty")
		}
		if firstCallLines[1] != "" {
			t.Errorf("second line was not empty. Got: %s", firstCallLines[1])
		}

		ctx.must(callCompiler(ctx, ctx.cfg, ctx.newCommand(gccX86_64, mainCc)))
		data, err = os.ReadFile(logFileName)
		if err != nil {
			t.Errorf("could not read the rusage log file. Error: %s", err)
		}
		secondCallLines := strings.Split(string(data), "\n")
		if len(secondCallLines) != 3 {
			t.Errorf("unexpected number of lines. Got: %s", secondCallLines)
		}
		if secondCallLines[0] != firstCallLines[0] {
			t.Errorf("first line was changed. Got: %s", secondCallLines[0])
		}
		if secondCallLines[1] == "" {
			t.Error("second line was empty")
		}
		if secondCallLines[2] != "" {
			t.Errorf("third line was not empty. Got: %s", secondCallLines[2])
		}
	})
}

func withLogRusageTestContext(t *testing.T, work func(ctx *testContext)) {
	withTestContext(t, func(ctx *testContext) {
		ctx.NoteTestWritesToUmask()

		ctx.env = []string{"TOOLCHAIN_RUSAGE_OUTPUT=" + filepath.Join(ctx.tempDir, "rusage.log")}
		work(ctx)
	})
}
