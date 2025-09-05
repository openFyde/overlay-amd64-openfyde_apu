// Copyright 2019 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"testing"
)

func TestAddNoMovbeFlagOnX86(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		cmd := ctx.must(callCompiler(ctx, ctx.cfg,
			ctx.newCommand(gccX86_64, mainCc)))
		if err := verifyArgOrder(cmd, mainCc, "-mno-movbe"); err != nil {
			t.Error(err)
		}
	})
}

func TestAddNoMovbeFlagOnI686(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		cmd := ctx.must(callCompiler(ctx, ctx.cfg,
			ctx.newCommand("./i686-cros-linux-gnu-gcc", mainCc)))
		if err := verifyArgOrder(cmd, mainCc, "-mno-movbe"); err != nil {
			t.Error(err)
		}
	})
}

func TestDoNotAddNoMovbeFlagOnArm(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		cmd := ctx.must(callCompiler(ctx, ctx.cfg,
			ctx.newCommand(gccArmV7, mainCc)))
		if err := verifyArgCount(cmd, 0, "-mno-movbe"); err != nil {
			t.Error(err)
		}
	})
}
