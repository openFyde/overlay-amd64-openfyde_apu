// Copyright 2025 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import "strings"

const (
	riscvExperimentalAckFlag  = "-D_CROSTC_USER_ACKNOWLEDGES_THAT_RISCV_IS_EXPERIMENTAL"
	riscvExperimentalEnvVar   = "CROSTC_USER_ACKNOWLEDGES_THAT_RISCV_IS_EXPERIMENTAL"
	riscvExperimentalUseError = "error: use of riscv is experimental. If you're not sure what " +
		"this implies, please reach out to chromeos-toolchain@google.com. If you've talked with " +
		"the toolchain team, pass '" + riscvExperimentalAckFlag + "' " +
		"to bypass this error, or set '" + riscvExperimentalEnvVar + "' in your " +
		"environment"
)

func isRiscvBuildWithoutAckFlag(env env, builder *commandBuilder) bool {
	// This is only relevant for CrOS.
	if builder.cfg.isAndroidWrapper {
		return false
	}

	if !strings.HasPrefix(builder.target.arch, "riscv") {
		return false
	}

	if _, ok := env.getenv(riscvExperimentalEnvVar); ok {
		return false
	}

	for _, arg := range builder.args {
		if arg.value == riscvExperimentalAckFlag {
			return false
		}
	}
	return true
}
