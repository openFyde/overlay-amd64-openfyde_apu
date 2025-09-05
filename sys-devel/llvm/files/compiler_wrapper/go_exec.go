// Copyright 2020 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

//go:build !libc_exec
// +build !libc_exec

package main

// Implement exec for users that don't need to dynamically link with glibc
// See b/144783188 and libc_exec.go.

func execCmd(env env, cmd *command) error {
	return execCmdWithoutLibc(env, cmd)
}
