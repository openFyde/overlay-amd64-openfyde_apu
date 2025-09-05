// Copyright 2019 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"path/filepath"
	"strings"
)

const skipSysrootAutodetectionFlag = "--cros-skip-wrapper-sysroot-autodetection"

func processSysrootFlag(builder *commandBuilder) {
	hadSkipSysrootMagicFlag := false
	fromUser := false
	userSysroot := ""
	builder.transformArgs(func(arg builderArg) string {
		switch {
		// In rare cases (e.g., glibc), we want all sysroot autodetection logic to be
		// disabled. This flag can be passed to disable that.
		case arg.value == skipSysrootAutodetectionFlag:
			hadSkipSysrootMagicFlag = true
			return ""

		case arg.fromUser && strings.HasPrefix(arg.value, "--sysroot="):
			fromUser = true
			sysrootArg := strings.Split(arg.value, "=")
			if len(sysrootArg) == 2 {
				userSysroot = sysrootArg[1]
			}
			return arg.value

		default:
			return arg.value
		}
	})

	if hadSkipSysrootMagicFlag {
		return
	}

	sysroot, syrootPresent := builder.env.getenv("SYSROOT")
	if syrootPresent {
		builder.updateEnv("SYSROOT=")
	}
	if sysroot == "" {
		// Use the bundled sysroot by default.
		sysroot = filepath.Join(builder.rootPath, "usr", builder.target.target)
	}
	if !fromUser {
		builder.addPreUserArgs("--sysroot=" + sysroot)
	} else {
		sysroot = userSysroot
	}

	libdir := "-L" + sysroot + "/usr/lib"
	if strings.Contains(builder.target.target, "64") {
		libdir += "64"
	}
	builder.addPostUserArgs(libdir)
}
