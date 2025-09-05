// Copyright 2019 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"path"
	"regexp"
	"testing"
)

func TestOmitSysrootGivenSysrootSuppressionFlag(t *testing.T) {
	escapedAutodetectionFlag := regexp.QuoteMeta(skipSysrootAutodetectionFlag)
	withTestContext(t, func(ctx *testContext) {
		runWithCompiler := func(compiler string) {
			cmd := ctx.must(callCompiler(ctx, ctx.cfg,
				ctx.newCommand(compiler, skipSysrootAutodetectionFlag, mainCc)))
			if err := verifyArgOrder(cmd, mainCc); err != nil {
				t.Error(err)
			}
			if err := verifyArgCount(cmd, 0, "--sysroot.*"); err != nil {
				t.Error(err)
			}
			if err := verifyArgCount(cmd, 0, "-L.*"); err != nil {
				t.Error(err)
			}
			if err := verifyArgCount(cmd, 0, escapedAutodetectionFlag); err != nil {
				t.Error(err)
			}
		}

		runWithCompiler(gccX86_64)
		runWithCompiler(clangX86_64)
	})
}

func TestOmitSysrootGivenUserDefinedSysroot(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		runWithCompiler := func(compiler string) {
			cmd := ctx.must(callCompiler(ctx, ctx.cfg,
				ctx.newCommand(compiler, "--sysroot=/somepath", mainCc)))
			if err := verifyArgOrder(cmd, "--sysroot=/somepath", mainCc); err != nil {
				t.Error(err)
			}
			if err := verifyArgCount(cmd, 1, "--sysroot.*"); err != nil {
				t.Error(err)
			}
		}

		runWithCompiler(gccX86_64)
		runWithCompiler(clangX86_64)
	})
}

func TestSetSysrootFlagFromEnv(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		ctx.env = []string{"SYSROOT=/envpath"}
		cmd := ctx.must(callCompiler(ctx, ctx.cfg,
			ctx.newCommand(gccX86_64, mainCc)))
		if err := verifyEnvUpdate(cmd, "SYSROOT="); err != nil {
			t.Error(err)
		}
		if err := verifyArgOrder(cmd, "--sysroot=/envpath", mainCc); err != nil {
			t.Error(err)
		}
	})
}

func TestClearEmptySysrootFlagInEnv(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		ctx.env = []string{"SYSROOT="}
		cmd := ctx.must(callCompiler(ctx, ctx.cfg,
			ctx.newCommand(gccX86_64, mainCc)))
		if err := verifyEnvUpdate(cmd, "SYSROOT="); err != nil {
			t.Error(err)
		}
		if err := verifyArgOrder(cmd, "--sysroot=.*/x86_64-cros-linux-gnu", mainCc); err != nil {
			t.Error(err)
		}
	})
}

func TestSetSysrootRelativeToWrapperPath(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		ctx.cfg.gccRootRelPath = "somepath"
		cmd := ctx.must(callCompiler(ctx, ctx.cfg,
			ctx.newCommand(gccX86_64, mainCc)))
		if err := verifyArgOrder(cmd,
			"--sysroot="+ctx.tempDir+"/somepath/usr/x86_64-cros-linux-gnu", mainCc); err != nil {
			t.Error(err)
		}
	})
}

func TestSetSysrootRelativeToSymlinkedWrapperPath(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		ctx.cfg.gccRootRelPath = "somepath"
		linkedWrapperPath := path.Join(ctx.tempDir, "a/linked/path/x86_64-cros-linux-gnu-gcc")
		ctx.symlink(path.Join(ctx.tempDir, gccX86_64), linkedWrapperPath)

		cmd := ctx.must(callCompiler(ctx, ctx.cfg,
			ctx.newCommand(linkedWrapperPath, mainCc)))
		if err := verifyArgOrder(cmd,
			"--sysroot="+ctx.tempDir+"/somepath/usr/x86_64-cros-linux-gnu", mainCc); err != nil {
			t.Error(err)
		}
	})
}
