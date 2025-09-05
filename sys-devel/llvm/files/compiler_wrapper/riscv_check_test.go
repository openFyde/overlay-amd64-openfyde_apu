// Copyright 2025 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"strings"
	"testing"
)

func TestRiscvBuildWithAckFlag(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		// Just make sure no errors are raised.
		ctx.must(callCompiler(ctx, ctx.cfg, ctx.newCommand(clangRiscv, riscvExperimentalAckFlag, mainCc)))
	})
}

func TestRiscvBuildWithEnvVar(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		ctx.env = append(ctx.env, riscvExperimentalEnvVar+"=")
		// Just make sure no errors are raised.
		ctx.must(callCompiler(ctx, ctx.cfg, ctx.newCommand(clangRiscv, riscvExperimentalAckFlag, mainCc)))
	})
}

func TestRiscvBuildWithoutAckFlagOrEnvVar(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		exitCode := callCompiler(ctx, ctx.cfg, ctx.newCommand(clangRiscv, mainCc))
		if exitCode == 0 {
			t.Errorf("riscv-clang without ack flag or env var should've exited with a non-zero code")
		}

		stderr := ctx.stderrBuffer.String()
		if !strings.Contains(stderr, riscvExperimentalUseError) {
			t.Errorf("riscv-clang without ack flag didn't produce stderr with error; got: %q", stderr)
		}
	})
}

func TestRiscvBuildWithoutAckFlagOnAndroid(t *testing.T) {
	withAndroidTestContext(t, func(ctx *testContext) {
		// Just make sure no errors are raised. (Yes, this technically uses a cros triple on android;
		// it's close enough).
		ctx.must(callCompiler(ctx, ctx.cfg, ctx.newCommand(clangRiscv, mainCc)))
	})
}
