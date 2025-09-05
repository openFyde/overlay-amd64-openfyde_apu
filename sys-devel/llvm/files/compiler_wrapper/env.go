// Copyright 2019 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"syscall"
	"time"
)

const artifactsTmpDirEnvName = "CROS_ARTIFACTS_TMP_DIR"

type env interface {
	umask(int) int
	getenv(key string) (string, bool)
	environ() []string
	getwd() string
	stdin() io.Reader
	stdout() io.Writer
	stderr() io.Writer
	run(cmd *command, stdin io.Reader, stdout io.Writer, stderr io.Writer) error
	runWithTimeout(cmd *command, duration time.Duration) error
	exec(cmd *command) error
}

type processEnv struct {
	wd string
}

func newProcessEnv() (env, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, wrapErrorwithSourceLocf(err, "failed to read working directory")
	}

	// Note: On Linux, Getwd may resolve to /proc/self/cwd, since it checks the PWD environment
	// variable. We need to read the link to get the actual working directory. We can't always
	// do this as we are calculating the path to clang, since following a symlinked cwd first
	// would make this calculation invalid.
	//
	// FIXME(gbiv): It's not clear why always Readlink()ing here an issue. crrev.com/c/1764624
	// might provide helpful context?
	if wd == "/proc/self/cwd" {
		wd, err = os.Readlink(wd)
		if err != nil {
			return nil, wrapErrorwithSourceLocf(err, "resolving /proc/self/cwd")
		}
	}

	return &processEnv{wd: wd}, nil
}

var _ env = (*processEnv)(nil)

func (env *processEnv) umask(newmask int) (oldmask int) {
	return syscall.Umask(newmask)
}

func (env *processEnv) getenv(key string) (string, bool) {
	return os.LookupEnv(key)
}

func (env *processEnv) environ() []string {
	return os.Environ()
}

func (env *processEnv) getwd() string {
	return env.wd
}

func (env *processEnv) stdin() io.Reader {
	return os.Stdin
}

func (env *processEnv) stdout() io.Writer {
	return os.Stdout
}

func (env *processEnv) stderr() io.Writer {
	return os.Stderr
}

func (env *processEnv) exec(cmd *command) error {
	return execCmd(env, cmd)
}

func (env *processEnv) runWithTimeout(cmd *command, duration time.Duration) error {
	return runCmdWithTimeout(env, cmd, duration)
}

func (env *processEnv) run(cmd *command, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	return runCmd(env, cmd, stdin, stdout, stderr)
}

type commandRecordingEnv struct {
	env
	stdinReader io.Reader
	cmdResults  []*commandResult
}
type commandResult struct {
	Cmd      *command `json:"cmd"`
	Stdout   string   `json:"stdout,omitempty"`
	Stderr   string   `json:"stderr,omitempty"`
	ExitCode int      `json:"exitcode,omitempty"`
}

var _ env = (*commandRecordingEnv)(nil)

func (env *commandRecordingEnv) stdin() io.Reader {
	return env.stdinReader
}

func (env *commandRecordingEnv) exec(cmd *command) error {
	// Note: We treat exec the same as run so that we can do work
	// after the call.
	return env.run(cmd, env.stdin(), env.stdout(), env.stderr())
}

func (env *commandRecordingEnv) runWithTimeout(cmd *command, duration time.Duration) error {
	return runCmdWithTimeout(env, cmd, duration)
}

func (env *commandRecordingEnv) run(cmd *command, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	stdoutBuffer := &bytes.Buffer{}
	stderrBuffer := &bytes.Buffer{}
	err := env.env.run(cmd, stdin, io.MultiWriter(stdout, stdoutBuffer), io.MultiWriter(stderr, stderrBuffer))
	if exitCode, ok := getExitCode(err); ok {
		env.cmdResults = append(env.cmdResults, &commandResult{
			Cmd:      cmd,
			Stdout:   stdoutBuffer.String(),
			Stderr:   stderrBuffer.String(),
			ExitCode: exitCode,
		})
	}
	return err
}

type printingEnv struct {
	env
}

var _ env = (*printingEnv)(nil)

func (env *printingEnv) exec(cmd *command) error {
	printCmd(env, cmd)
	return env.env.exec(cmd)
}

func (env *printingEnv) runWithTimeout(cmd *command, duration time.Duration) error {
	printCmd(env, cmd)
	return env.env.runWithTimeout(cmd, duration)
}

func (env *printingEnv) run(cmd *command, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	printCmd(env, cmd)
	return env.env.run(cmd, stdin, stdout, stderr)
}

func printCmd(env env, cmd *command) {
	fmt.Fprintf(env.stderr(), "cd '%s' &&", env.getwd())
	if len(cmd.EnvUpdates) > 0 {
		fmt.Fprintf(env.stderr(), " env '%s'", strings.Join(cmd.EnvUpdates, "' '"))
	}
	fmt.Fprintf(env.stderr(), " '%s'", getAbsCmdPath(env, cmd))
	if len(cmd.Args) > 0 {
		fmt.Fprintf(env.stderr(), " '%s'", strings.Join(cmd.Args, "' '"))
	}
	io.WriteString(env.stderr(), "\n")
}

func getCompilerArtifactsDir(env env) string {
	const defaultArtifactDir = "/tmp"
	value, _ := env.getenv(artifactsTmpDirEnvName)
	if value == "" {
		fmt.Fprintf(env.stdout(), "$%s is not set, artifacts will be written to %s", artifactsTmpDirEnvName, defaultArtifactDir)
		return defaultArtifactDir
	}
	return value

}
