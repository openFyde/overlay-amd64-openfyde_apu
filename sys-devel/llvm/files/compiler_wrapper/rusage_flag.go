// Copyright 2019 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

func getRusageLogFilename(env env) string {
	value, _ := env.getenv("TOOLCHAIN_RUSAGE_OUTPUT")
	return value
}

func isRusageEnabled(env env) bool {
	return getRusageLogFilename(env) != ""
}

func lockFileExclusive(fd uintptr) error {
	maxTries := 100
	for i := 0; i < maxTries; i++ {
		const seekSet = 0
		err := syscall.FcntlFlock(fd, syscall.F_SETLKW, &syscall.Flock_t{
			Type:   syscall.F_WRLCK,
			Whence: seekSet,
			Start:  0,
			Len:    0,
		})
		if err == nil {
			return nil
		}
		if err != syscall.EINTR {
			return fmt.Errorf("locking file: %v", err)
		}
	}
	return fmt.Errorf("locking file failed after %d tries", maxTries)
}

type rusageLog struct {
	ExitCode         int      `json:"exit_code"`
	ElapsedRealTime  float64  `json:"elapsed_real_time"`
	ElapsedUserTime  float64  `json:"elapsed_user_time"`
	ElapsedSysTime   float64  `json:"elapsed_sys_time"`
	MaxMemUsed       int64    `json:"max_mem_used"`
	Compiler         string   `json:"compiler"`
	CompilerArgs     []string `json:"compiler_args"`
	WorkingDirectory string   `json:"working_directory"`
}

func removeRusageFromCommand(compilerCmd *command) *command {
	return &command{
		Path:       compilerCmd.Path,
		Args:       compilerCmd.Args,
		EnvUpdates: append(compilerCmd.EnvUpdates, "TOOLCHAIN_RUSAGE_OUTPUT="),
	}
}

// maybeCaptureRusage captures rusage for execution of action()
// unless action returns an error or logFileName is ""
// a function is returned which saves the rusage log data at logFileName unless logFileName is ""
// an error is returned if action returns an error, or rusage commands in syscall fails
func maybeCaptureRusage(env env, compilerCmd *command, action func(willLogRusage bool) error) (maybeCommitToFile func(exitCode int) error, err error) {
	logFileName := getRusageLogFilename(env)
	willLogRusage := isRusageEnabled(env)
	if !willLogRusage {
		if err := action(willLogRusage); err != nil {
			return nil, err
		}
		return func(int) error { return nil }, nil
	}

	rusageBefore := syscall.Rusage{}
	if err := syscall.Getrusage(syscall.RUSAGE_CHILDREN, &rusageBefore); err != nil {
		return nil, fmt.Errorf("getting initial rusage: %v", err)
	}
	startTime := time.Now()

	if err := action(willLogRusage); err != nil {
		return nil, err
	}

	elapsedRealTime := time.Since(startTime)
	rusageAfter := syscall.Rusage{}
	if err := syscall.Getrusage(syscall.RUSAGE_CHILDREN, &rusageAfter); err != nil {
		return nil, fmt.Errorf("getting final rusage: %v", err)
	}

	return func(exitCode int) error {
		elapsedSysTime := time.Duration(rusageAfter.Stime.Nano()-rusageBefore.Stime.Nano()) * time.Nanosecond
		elapsedUserTime := time.Duration(rusageAfter.Utime.Nano()-rusageBefore.Utime.Nano()) * time.Nanosecond
		// Note: We assume that the compiler takes more heap than any other
		// subcommands that we might have executed before.
		maxMemUsed := rusageAfter.Maxrss
		absCompilerPath := getAbsCmdPath(env, compilerCmd)

		// We need to temporarily set umask to 0 to ensure 777 permissions are actually 777
		// This effects builderbots in particular
		oldMask := env.umask(0)
		defer env.umask(oldMask)

		// We want to know what package is being compiled. The working directory gives us a good clue.
		cwd, err := os.Getwd()
		if err != nil {
			return wrapErrorwithSourceLocf(err, "error getting working directory for rusage log")
		}

		if err := os.MkdirAll(filepath.Dir(logFileName), 0777); err != nil {
			return wrapErrorwithSourceLocf(err, "error creating rusage log directory %s", logFileName)
		}

		timeUnit := float64(time.Second)

		logEntry := rusageLog{
			ExitCode:         exitCode,
			ElapsedRealTime:  float64(elapsedRealTime) / timeUnit,
			ElapsedUserTime:  float64(elapsedUserTime) / timeUnit,
			ElapsedSysTime:   float64(elapsedSysTime) / timeUnit,
			MaxMemUsed:       maxMemUsed,
			Compiler:         absCompilerPath,
			CompilerArgs:     compilerCmd.Args,
			WorkingDirectory: cwd,
		}

		// Note: using file mode 0666 so that a root-created log is writable by others.
		logFile, err := os.OpenFile(logFileName, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
		if err != nil {
			return wrapErrorwithSourceLocf(err, "creating rusage logfile %s", logFileName)
		}

		// O_APPEND's atomicity guarantees are only for writes up to a certain size. If we don't
		// lock the file, we might end up with corrupted records.
		//
		// Note that Close()'ing the file releases all associated locks.
		if err := lockFileExclusive(logFile.Fd()); err != nil {
			_ = logFile.Close()
			return wrapErrorwithSourceLocf(err, "locking rusage logfile %s: %v", logFileName, err)
		}

		if err := json.NewEncoder(logFile).Encode(logEntry); err != nil {
			_ = logFile.Close()
			return wrapErrorwithSourceLocf(err, "converting rusage logfile entry to JSON %v", logEntry)
		}

		closeErr := logFile.Close()
		if err != nil {
			return wrapErrorwithSourceLocf(err, "writing to rusage logfile %s: %v", logFileName, err)
		}
		if closeErr != nil {
			return wrapErrorwithSourceLocf(err, "closing rusage logfile %s: %v", logFileName, closeErr)
		}

		return nil
	}, nil
}
