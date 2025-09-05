// Copyright 2019 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

const prebuiltCompilerPathKey = "ANDROID_LLVM_PREBUILT_COMPILER_PATH"

func shouldCompileWithFallback(env env) bool {
	value, _ := env.getenv(prebuiltCompilerPathKey)
	return value != ""
}

// FIXME: Deduplicate this logic with the logic for FORCE_DISABLE_WERROR
// (the logic here is from Android, the logic for FORCE_DISABLE_WERROR is from ChromeOS)
func compileWithFallback(env env, cfg *config, originalCmd *command, absWrapperPath string) (exitCode int, err error) {
	firstCmd := &command{
		Path:       originalCmd.Path,
		Args:       originalCmd.Args,
		EnvUpdates: originalCmd.EnvUpdates,
	}
	// We only want to pass extra flags to clang and clang++.
	if base := filepath.Base(originalCmd.Path); base == "clang-real" || base == "clang++-real" {
		// We may introduce some new warnings after rebasing and we need to
		// disable them before we fix those warnings.
		extraArgs, _ := env.getenv("ANDROID_LLVM_FALLBACK_DISABLED_WARNINGS")
		firstCmd.Args = append(
			append(firstCmd.Args, "-fno-color-diagnostics"),
			strings.Split(extraArgs, " ")...,
		)
	}

	getStdin, err := prebufferStdinIfNeeded(env, firstCmd)
	if err != nil {
		return 0, wrapErrorwithSourceLocf(err, "prebuffering stdin: %v", err)
	}

	firstCmdStderrBuffer := &bytes.Buffer{}
	firstCmdExitCode, err := wrapSubprocessErrorWithSourceLoc(firstCmd,
		env.run(firstCmd, getStdin(), env.stdout(), io.MultiWriter(env.stderr(), firstCmdStderrBuffer)))
	if err != nil {
		return 0, err
	}

	if firstCmdExitCode == 0 {
		return 0, nil
	}
	stderrRedirectPath, _ := env.getenv("ANDROID_LLVM_STDERR_REDIRECT")
	f, err := os.OpenFile(stderrRedirectPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return 0, wrapErrorwithSourceLocf(err, "error opening stderr file %s", stderrRedirectPath)
	}
	lockSuccess := false
	for i := 0; i < 30; i++ {
		err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if err == nil {
			lockSuccess = true
			break
		}
		if errno, ok := err.(syscall.Errno); ok {
			if errno == syscall.EAGAIN || errno == syscall.EACCES {
				time.Sleep(500 * time.Millisecond)
				err = nil
			}
		}
		if err != nil {
			return 0, wrapErrorwithSourceLocf(err, "error waiting to lock file %s", stderrRedirectPath)
		}
	}
	if !lockSuccess {
		return 0, wrapErrorwithSourceLocf(err, "timeout waiting to lock file %s", stderrRedirectPath)
	}
	w := bufio.NewWriter(f)
	w.WriteString("==================COMMAND:====================\n")
	fmt.Fprintf(w, "%s %s\n\n", firstCmd.Path, strings.Join(firstCmd.Args, " "))
	firstCmdStderrBuffer.WriteTo(w)
	w.WriteString("==============================================\n\n")
	if err := w.Flush(); err != nil {
		return 0, wrapErrorwithSourceLocf(err, "unable to write to file %s", stderrRedirectPath)
	}
	if err := f.Close(); err != nil {
		return 0, wrapErrorwithSourceLocf(err, "error closing file %s", stderrRedirectPath)
	}

	prebuiltCompilerPath, _ := env.getenv(prebuiltCompilerPathKey)
	fallbackCmd := &command{
		Path: filepath.Join(prebuiltCompilerPath, filepath.Base(absWrapperPath)),
		// Don't use extra args added (from ANDROID_LLVM_FALLBACK_DISABLED_WARNINGS) for clang and
		// clang++ above.  They may not be recognized by the fallback clang.
		Args: originalCmd.Args,
		// Delete prebuiltCompilerPathKey so the fallback doesn't keep
		// calling itself in case of an error.
		EnvUpdates: append(originalCmd.EnvUpdates, prebuiltCompilerPathKey+"="),
	}
	return wrapSubprocessErrorWithSourceLoc(fallbackCmd,
		env.run(fallbackCmd, getStdin(), env.stdout(), env.stderr()))
}
