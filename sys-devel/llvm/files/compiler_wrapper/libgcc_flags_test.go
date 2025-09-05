// Copyright 2021 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"testing"
)

func TestDefaultStaticLibGCC(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		runWithCompiler := func(compiler string) {
			cmd := ctx.must(callCompiler(ctx, ctx.cfg,
				ctx.newCommand(compiler, mainCc)))
			if err := verifyArgCount(cmd, 1, "-static-libgcc"); err != nil {
				t.Error(err)
			}
		}

		runWithCompiler(gccX86_64)
		runWithCompiler(clangX86_64)
	})
}

func TestKeepStaticLibGCCWithUserArgs(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		runWithCompiler := func(compiler string) {
			cmd := ctx.must(callCompiler(ctx, ctx.cfg,
				ctx.newCommand(compiler, "-static-libgcc", mainCc)))
			if err := verifyArgOrder(cmd, "-static-libgcc", mainCc); err != nil {
				t.Error(err)
			}
		}

		runWithCompiler(gccX86_64)
		runWithCompiler(clangX86_64)
	})
}

func TestNoAddedStaticLibGCCWithSharedLibGCC(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		runWithCompiler := func(compiler string) {
			cmd := ctx.must(callCompiler(ctx, ctx.cfg,
				ctx.newCommand(compiler, "-shared-libgcc", mainCc)))
			if err := verifyArgCount(cmd, 0, "-static-libgcc"); err != nil {
				t.Error(err)
			}
			if err := verifyArgCount(cmd, 1, "-shared-libgcc"); err != nil {
				t.Error(err)
			}
		}

		runWithCompiler(gccX86_64)
		runWithCompiler(clangX86_64)
	})
}
