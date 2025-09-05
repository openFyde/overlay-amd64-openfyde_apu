// Copyright 2019 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Attention: The tests in this file execute the test binary again with the `-run` flag.
// This is needed as they want to test an `exec`, which terminates the test process.
var internalexececho = flag.Bool("internalexececho", false, "internal flag used for tests that exec")

func TestProcessEnvExecPathAndArgs(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		if *internalexececho {
			execEcho(ctx, &command{
				Path: "some_binary",
				Args: []string{"arg1", "arg2"},
			})
			return
		}
		logLines := forkAndReadEcho(ctx)
		if !strings.HasSuffix(logLines[0], "/some_binary arg1 arg2") {
			t.Errorf("incorrect path or args: %s", logLines[0])
		}
	})
}

func TestProcessEnvExecAddEnv(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		if *internalexececho {
			execEcho(ctx, &command{
				Path:       "some_binary",
				EnvUpdates: []string{"ABC=xyz"},
			})
			return
		}

		logLines := forkAndReadEcho(ctx)
		for _, ll := range logLines {
			if ll == "ABC=xyz" {
				return
			}
		}
		t.Errorf("could not find new env variable: %s", logLines)
	})
}

func TestProcessEnvExecUpdateEnv(t *testing.T) {
	if os.Getenv("PATH") == "" {
		t.Fatal("no PATH environment variable found!")
	}
	withTestContext(t, func(ctx *testContext) {
		if *internalexececho {
			execEcho(ctx, &command{
				Path:       "some_binary",
				EnvUpdates: []string{"PATH=xyz"},
			})
			return
		}
		logLines := forkAndReadEcho(ctx)
		for _, ll := range logLines {
			if ll == "PATH=xyz" {
				return
			}
		}
		t.Errorf("could not find updated env variable: %s", logLines)
	})
}

func TestProcessEnvExecDeleteEnv(t *testing.T) {
	if os.Getenv("PATH") == "" {
		t.Fatal("no PATH environment variable found!")
	}
	withTestContext(t, func(ctx *testContext) {
		if *internalexececho {
			execEcho(ctx, &command{
				Path:       "some_binary",
				EnvUpdates: []string{"PATH="},
			})
			return
		}
		logLines := forkAndReadEcho(ctx)
		for _, ll := range logLines {
			if strings.HasPrefix(ll, "PATH=") {
				t.Errorf("path env was not removed: %s", ll)
			}
		}
	})
}

func TestProcessEnvRunCmdPathAndArgs(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		cmd := &command{
			Path: "some_binary",
			Args: []string{"arg1", "arg2"},
		}
		logLines := runAndEcho(ctx, cmd)
		if !strings.HasSuffix(logLines[0], "/some_binary arg1 arg2") {
			t.Errorf("incorrect path or args: %s", logLines[0])
		}
	})
}

func TestProcessEnvRunCmdAddEnv(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		cmd := &command{
			Path:       "some_binary",
			EnvUpdates: []string{"ABC=xyz"},
		}
		logLines := runAndEcho(ctx, cmd)
		for _, ll := range logLines {
			if ll == "ABC=xyz" {
				return
			}
		}
		t.Errorf("could not find new env variable: %s", logLines)
	})
}

func TestProcessEnvRunCmdUpdateEnv(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		if os.Getenv("PATH") == "" {
			t.Fatal("no PATH environment variable found!")
		}
		cmd := &command{
			Path:       "some_binary",
			EnvUpdates: []string{"PATH=xyz"},
		}
		logLines := runAndEcho(ctx, cmd)
		for _, ll := range logLines {
			if ll == "PATH=xyz" {
				return
			}
		}
		t.Errorf("could not find updated env variable: %s", logLines)
	})
}

func TestProcessEnvRunCmdDeleteEnv(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		if os.Getenv("PATH") == "" {
			t.Fatal("no PATH environment variable found!")
		}
		cmd := &command{
			Path:       "some_binary",
			EnvUpdates: []string{"PATH="},
		}
		logLines := runAndEcho(ctx, cmd)
		for _, ll := range logLines {
			if strings.HasPrefix(ll, "PATH=") {
				t.Errorf("path env was not removed: %s", ll)
			}
		}
	})
}

func TestRunWithTimeoutRunsTheGivenProcess(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		env, err := newProcessEnv()
		if err != nil {
			t.Fatalf("Unexpected error making new process env: %v", err)
		}

		tempFile := path.Join(ctx.tempDir, "some_file")
		cmd := &command{
			Path: "touch",
			Args: []string{tempFile},
		}
		if err := env.runWithTimeout(cmd, time.Second*120); err != nil {
			t.Fatalf("Unexpected error touch'ing %q: %v", tempFile, err)
		}

		// This should be fine, since `touch` should've created the file.
		if _, err := os.Stat(tempFile); err != nil {
			t.Errorf("Stat'ing temp file at %q failed: %v", tempFile, err)
		}
	})
}

func TestRunWithTimeoutReturnsErrorOnTimeout(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		env, err := newProcessEnv()
		if err != nil {
			t.Fatalf("Unexpected error making new process env: %v", err)
		}

		cmd := &command{
			Path: "sleep",
			Args: []string{"30"},
		}

		err = env.runWithTimeout(cmd, 100*time.Millisecond)
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("Expected context.DeadlineExceeded after `sleep` timed out; got error: %v", err)
		}
	})
}

func TestNewProcessEnvResolvesPwdAwayProperly(t *testing.T) {
	// This test cannot be t.Parallel(), since it modifies our environment.
	const envPwd = "PWD"

	oldEnvPwd := os.Getenv(envPwd)
	defer func() {
		if oldEnvPwd == "" {
			os.Unsetenv(envPwd)
		} else {
			os.Setenv(envPwd, oldEnvPwd)
		}
	}()

	os.Unsetenv(envPwd)

	initialWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed getting working directory: %v", err)
	}
	if initialWd == "/proc/self/cwd" {
		t.Fatalf("Working directory should never be %q when env is unset", initialWd)
	}

	defer func() {
		if err := os.Chdir(initialWd); err != nil {
			t.Errorf("Changing back to %q failed: %v", initialWd, err)
		}
	}()

	tempDir, err := os.MkdirTemp("", "wrapper_env_test")
	if err != nil {
		t.Fatalf("Failed making temp dir: %v", err)
	}

	// Nothing we can do if this breaks, unfortunately.
	defer os.RemoveAll(tempDir)

	tempDirLink := tempDir + ".symlink"
	if err := os.Symlink(tempDir, tempDirLink); err != nil {
		t.Fatalf("Failed creating symlink %q => %q: %v", tempDirLink, tempDir, err)
	}

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed chdir'ing to tempdir at %q: %v", tempDirLink, err)
	}

	if err := os.Setenv(envPwd, tempDirLink); err != nil {
		t.Fatalf("Failed setting pwd to tempdir at %q: %v", tempDirLink, err)
	}

	// Ensure that we don't resolve symlinks if they're present in our CWD somehow, except for
	// /proc/self/cwd, which tells us nothing about where we are.
	env, err := newProcessEnv()
	if err != nil {
		t.Fatalf("Failed making a new env: %v", err)
	}

	if wd := env.getwd(); wd != tempDirLink {
		t.Errorf("Environment setup had a wd of %q; wanted %q", wd, tempDirLink)
	}

	const cwdLink = "/proc/self/cwd"
	if err := os.Setenv(envPwd, cwdLink); err != nil {
		t.Fatalf("Failed setting pwd to /proc/self/cwd: %v", err)
	}

	env, err = newProcessEnv()
	if err != nil {
		t.Fatalf("Failed making a new env: %v", err)
	}

	if wd := env.getwd(); wd != tempDir {
		t.Errorf("Environment setup had a wd of %q; wanted %q", cwdLink, tempDir)
	}
}

func execEcho(ctx *testContext, cmd *command) {
	env := &processEnv{}
	err := env.exec(createEcho(ctx, cmd))
	if err != nil {
		os.Stderr.WriteString(err.Error())
	}
	os.Exit(1)
}

func forkAndReadEcho(ctx *testContext) []string {
	testBin, err := os.Executable()
	if err != nil {
		ctx.t.Fatalf("unable to read the executable: %s", err)
	}

	subCmd := exec.Command(testBin, "-internalexececho", "-test.run="+ctx.t.Name())
	output, err := subCmd.CombinedOutput()
	if err != nil {
		ctx.t.Fatalf("error calling test binary again for exec: %s", err)
	}
	return strings.Split(string(output), "\n")
}

func runAndEcho(ctx *testContext, cmd *command) []string {
	env, err := newProcessEnv()
	if err != nil {
		ctx.t.Fatalf("creation of process env failed: %s", err)
	}
	buffer := bytes.Buffer{}
	if err := env.run(createEcho(ctx, cmd), nil, &buffer, &buffer); err != nil {
		ctx.t.Fatalf("run failed: %s", err)
	}
	return strings.Split(buffer.String(), "\n")
}

func createEcho(ctx *testContext, cmd *command) *command {
	content := `
/bin/echo "$0" "$@"
/usr/bin/env
`
	fullPath := filepath.Join(ctx.tempDir, cmd.Path)
	ctx.writeFile(fullPath, content)
	// Note: Using a self executable wrapper does not work due to a race condition
	// on unix systems. See https://github.com/golang/go/issues/22315
	return &command{
		Path:       "bash",
		Args:       append([]string{fullPath}, cmd.Args...),
		EnvUpdates: cmd.EnvUpdates,
	}
}
