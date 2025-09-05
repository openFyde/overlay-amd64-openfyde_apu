// Copyright 2025 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"os/exec"
	"syscall"
)

// Implementation of execCmd that does not use libc. This is used to avoid
// crashes (b/144783188 and b/406369220), but can only be used when the
// Portage sandbox is not in use. Inside the Portage sandbox, use
// execCmd from libc_exec, instead.
func execCmdWithoutLibc(env env, cmd *command) error {
	execCmd := exec.Command(cmd.Path, cmd.Args...)
	mergedEnv := mergeEnvValues(env.environ(), cmd.EnvUpdates)

	ret := syscall.Exec(execCmd.Path, execCmd.Args, mergedEnv)
	return newErrorwithSourceLocf("exec error: %v", ret)
}
