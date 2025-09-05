// Copyright 2019 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"testing"
)

func TestAddStrongStackProtectorFlag(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		initStackProtectorStrongConfig(ctx.cfg)
		cmd := ctx.must(callCompiler(ctx, ctx.cfg,
			ctx.newCommand(gccX86_64, mainCc)))
		if err := verifyArgOrder(cmd, "-fstack-protector-strong", mainCc); err != nil {
			t.Error(err)
		}
	})
}

func TestAddNoStackProtectorFlagWhenKernelDefined(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		initStackProtectorStrongConfig(ctx.cfg)
		cmd := ctx.must(callCompiler(ctx, ctx.cfg,
			ctx.newCommand(gccX86_64, "-D__KERNEL__", mainCc)))
		if err := verifyArgOrder(cmd, "-fno-stack-protector", mainCc); err != nil {
			t.Error(err)
		}
	})
}

func TestOmitNoStackProtectorFlagWhenAlreadyInCommonFlags(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		ctx.cfg.commonFlags = []string{"-fno-stack-protector"}
		cmd := ctx.must(callCompiler(ctx, ctx.cfg,
			ctx.newCommand(gccX86_64, mainCc)))
		if err := verifyArgCount(cmd, 1, "-fno-stack-protector"); err != nil {
			t.Error(err)
		}
	})
}

func TestAddStrongStackProtectorFlagForEabiEvenIfNoStackProtectorGiven(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		initStackProtectorStrongConfig(ctx.cfg)
		cmd := ctx.must(callCompiler(ctx, ctx.cfg,
			ctx.newCommand(gccX86_64Eabi, "-fno-stack-protector", mainCc)))
		if err := verifyArgCount(cmd, 1, "-fstack-protector-strong"); err != nil {
			t.Error(err)
		}
	})
}

func initStackProtectorStrongConfig(cfg *config) {
	cfg.commonFlags = []string{"-fstack-protector-strong"}
}
