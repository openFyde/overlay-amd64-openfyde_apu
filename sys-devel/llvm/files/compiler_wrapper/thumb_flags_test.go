// Copyright 2019 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"testing"
)

func TestAddThumbFlagForArm(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		cmd := ctx.must(callCompiler(ctx, ctx.cfg,
			ctx.newCommand(gccArmV7, mainCc)))
		if err := verifyArgOrder(cmd, "-mthumb", mainCc); err != nil {
			t.Error(err)
		}

		cmd = ctx.must(callCompiler(ctx, ctx.cfg,
			ctx.newCommand(gccArmV8, mainCc)))
		if err := verifyArgOrder(cmd, "-mthumb", mainCc); err != nil {
			t.Error(err)
		}
	})
}

func TestOmitThumbFlagForNonArm(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		cmd := ctx.must(callCompiler(ctx, ctx.cfg,
			ctx.newCommand(gccX86_64, mainCc)))
		if err := verifyArgCount(cmd, 0, "-mthumb"); err != nil {
			t.Error(err)
		}
	})
}

func TestOmitThumbFlagForEabiArm(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		cmd := ctx.must(callCompiler(ctx, ctx.cfg,
			ctx.newCommand(gccArmV7Eabi, mainCc)))
		if err := verifyArgCount(cmd, 0, "-mthumb"); err != nil {
			t.Error(err)
		}

		cmd = ctx.must(callCompiler(ctx, ctx.cfg,
			ctx.newCommand(gccArmV8Eabi, mainCc)))
		if err := verifyArgCount(cmd, 0, "-mthumb"); err != nil {
			t.Error(err)
		}
	})
}

func TestRemoveNoOmitFramePointerFlagForArm(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		initNoOmitFramePointerConfig(ctx.cfg)

		cmd := ctx.must(callCompiler(ctx, ctx.cfg,
			ctx.newCommand(gccArmV7, mainCc)))
		if err := verifyArgCount(cmd, 0, "-fno-omit-frame-pointer"); err != nil {
			t.Error(err)
		}

		cmd = ctx.must(callCompiler(ctx, ctx.cfg,
			ctx.newCommand(gccArmV8, mainCc)))
		if err := verifyArgCount(cmd, 0, "-fno-omit-frame-pointer"); err != nil {
			t.Error(err)
		}
	})
}

func TestKeepNoOmitFramePointerFlagForNonArm(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		initNoOmitFramePointerConfig(ctx.cfg)

		cmd := ctx.must(callCompiler(ctx, ctx.cfg,
			ctx.newCommand(gccX86_64, mainCc)))
		if err := verifyArgCount(cmd, 1, "-fno-omit-frame-pointer"); err != nil {
			t.Error(err)
		}
	})
}

func TestKeepNoOmitFramePointerFlagForEabiArm(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		initNoOmitFramePointerConfig(ctx.cfg)

		cmd := ctx.must(callCompiler(ctx, ctx.cfg,
			ctx.newCommand(gccArmV7Eabi, mainCc)))
		if err := verifyArgCount(cmd, 1, "-fno-omit-frame-pointer"); err != nil {
			t.Error(err)
		}

		cmd = ctx.must(callCompiler(ctx, ctx.cfg,
			ctx.newCommand(gccArmV8Eabi, mainCc)))
		if err := verifyArgCount(cmd, 1, "-fno-omit-frame-pointer"); err != nil {
			t.Error(err)
		}
	})
}

func TestKeepNoOmitFramePointIfGivenByUser(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		cmd := ctx.must(callCompiler(ctx, ctx.cfg,
			ctx.newCommand(gccArmV7, "-fno-omit-frame-pointer", mainCc)))
		if err := verifyArgCount(cmd, 1, "-fno-omit-frame-pointer"); err != nil {
			t.Error(err)
		}
	})
}

func initNoOmitFramePointerConfig(cfg *config) {
	cfg.commonFlags = []string{"-fno-omit-frame-pointer"}
}
