// Copyright 2019 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"testing"
)

func TestFortifyIsKeptIfSanitizerIsTrivial(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		cmd := ctx.must(callCompiler(ctx, ctx.cfg,
			ctx.newCommand(gccX86_64, "-fsanitize=return", "-D_FORTIFY_SOURCE=1", mainCc)))
		if err := verifyArgCount(cmd, 1, "-D_FORTIFY_SOURCE=1"); err != nil {
			t.Error(err)
		}

		cmd = ctx.must(callCompiler(ctx, ctx.cfg,
			ctx.newCommand(gccX86_64, "-fsanitize=return,address", "-D_FORTIFY_SOURCE=1", mainCc)))
		if err := verifyArgCount(cmd, 0, "-D_FORTIFY_SOURCE=1"); err != nil {
			t.Error(err)
		}

		cmd = ctx.must(callCompiler(ctx, ctx.cfg,
			ctx.newCommand(gccX86_64, "-fsanitize=address,return", "-D_FORTIFY_SOURCE=1", mainCc)))
		if err := verifyArgCount(cmd, 0, "-D_FORTIFY_SOURCE=1"); err != nil {
			t.Error(err)
		}
	})
}

func TestFilterUnsupportedSanitizerFlagsIfSanitizeGiven(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		cmd := ctx.must(callCompiler(ctx, ctx.cfg,
			ctx.newCommand(gccX86_64, "-fsanitize=kernel-address", "-Wl,--no-undefined", mainCc)))
		if err := verifyArgCount(cmd, 0, "-Wl,--no-undefined"); err != nil {
			t.Error(err)
		}

		cmd = ctx.must(callCompiler(ctx, ctx.cfg,
			ctx.newCommand(gccX86_64, "-fsanitize=kernel-address", "-Wl,-z,defs", mainCc)))
		if err := verifyArgCount(cmd, 0, "-Wl,-z,defs"); err != nil {
			t.Error(err)
		}

		cmd = ctx.must(callCompiler(ctx, ctx.cfg,
			ctx.newCommand(gccX86_64, "-fsanitize=kernel-address", "-Wl,-z,defs,relro", mainCc)))
		if err := verifyArgCount(cmd, 1, "-Wl,relro"); err != nil {
			t.Error(err)
		}

		cmd = ctx.must(callCompiler(ctx, ctx.cfg,
			ctx.newCommand(gccX86_64, "-fsanitize=kernel-address", "-Wl,-z -Wl,defs", mainCc)))
		if err := verifyArgCount(cmd, 0, "-Wl,-z"); err != nil {
			t.Error(err)
		}
		if err := verifyArgCount(cmd, 0, "-Wl,defs"); err != nil {
			t.Error(err)
		}

		cmd = ctx.must(callCompiler(ctx, ctx.cfg,
			ctx.newCommand(gccX86_64, "-fsanitize=kernel-address", "-D_FORTIFY_SOURCE=1", mainCc)))
		if err := verifyArgCount(cmd, 0, "-D_FORTIFY_SOURCE=1"); err != nil {
			t.Error(err)
		}

		cmd = ctx.must(callCompiler(ctx, ctx.cfg,
			ctx.newCommand(gccX86_64, "-fsanitize=kernel-address", "-D_FORTIFY_SOURCE=2", mainCc)))
		if err := verifyArgCount(cmd, 0, "-D_FORTIFY_SOURCE=2"); err != nil {
			t.Error(err)
		}
	})
}

func TestAllFortifyEnableFlagsAreRemoved(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		flagsToRemove := []string{
			"-D_FORTIFY_SOURCE=1",
			"-D_FORTIFY_SOURCE=2",
			"-D_FORTIFY_SOURCE=3",
		}
		for _, fortifyFlag := range flagsToRemove {
			ctx.cfg.commonFlags = []string{fortifyFlag}
			cmd := ctx.must(callCompiler(ctx, ctx.cfg,
				ctx.newCommand(clangX86_64, "-fsanitize=kernel-address", mainCc)))
			if err := verifyArgCount(cmd, 0, fortifyFlag); err != nil {
				t.Errorf("Verifying FORTIFY flag %q is removed: %v", fortifyFlag, err)
			}
		}
	})
}

func TestFortifyDisableFlagsAreKept(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		flagsToKeep := []string{
			"-D_FORTIFY_SOURCE",
			"-D_FORTIFY_SOURCE=",
			"-D_FORTIFY_SOURCE=0",
		}
		for _, fortifyFlag := range flagsToKeep {
			ctx.cfg.commonFlags = []string{fortifyFlag}
			cmd := ctx.must(callCompiler(ctx, ctx.cfg,
				ctx.newCommand(clangX86_64, "-fsanitize=kernel-address", mainCc)))
			if err := verifyArgCount(cmd, 1, fortifyFlag); err != nil {
				t.Errorf("Verifying FORTIFY flag %q is kept: %v", fortifyFlag, err)
			}
		}
	})
}

func TestFilterUnsupportedDefaultSanitizerFlagsIfSanitizeGivenForClang(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		ctx.cfg.commonFlags = []string{"-D_FORTIFY_SOURCE=1"}
		cmd := ctx.must(callCompiler(ctx, ctx.cfg,
			ctx.newCommand(clangX86_64, "-fsanitize=kernel-address", mainCc)))
		if err := verifyArgCount(cmd, 0, "-D_FORTIFY_SOURCE=1"); err != nil {
			t.Error(err)
		}
	})
}

func TestKeepUnsupportedDefaultSanitizerFlagsIfSanitizeGivenForGcc(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		ctx.cfg.commonFlags = []string{"-D_FORTIFY_SOURCE=1"}
		cmd := ctx.must(callCompiler(ctx, ctx.cfg,
			ctx.newCommand(gccX86_64, "-fsanitize=kernel-address", mainCc)))
		if err := verifyArgCount(cmd, 1, "-D_FORTIFY_SOURCE=1"); err != nil {
			t.Error(err)
		}
	})
}

// TODO: This is a bug in the old wrapper to not filter
// non user args for gcc. Fix this once we don't compare to the old wrapper anymore.
func TestKeepSanitizerFlagsIfNoSanitizeGiven(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		cmd := ctx.must(callCompiler(ctx, ctx.cfg,
			ctx.newCommand(gccX86_64, "-Wl,--no-undefined", mainCc)))
		if err := verifyArgCount(cmd, 1, "-Wl,--no-undefined"); err != nil {
			t.Error(err)
		}

		cmd = ctx.must(callCompiler(ctx, ctx.cfg,
			ctx.newCommand(gccX86_64, "-Wl,-z,defs", mainCc)))
		if err := verifyArgCount(cmd, 1, "-Wl,-z,defs"); err != nil {
			t.Error(err)
		}

		cmd = ctx.must(callCompiler(ctx, ctx.cfg,
			ctx.newCommand(gccX86_64, "-Wl,-z", "-Wl,defs", mainCc)))
		if err := verifyArgCount(cmd, 1, "-Wl,-z"); err != nil {
			t.Error(err)
		}
		if err := verifyArgCount(cmd, 1, "-Wl,defs"); err != nil {
			t.Error(err)
		}

		cmd = ctx.must(callCompiler(ctx, ctx.cfg,
			ctx.newCommand(gccX86_64, "-D_FORTIFY_SOURCE=1", mainCc)))
		if err := verifyArgCount(cmd, 1, "-D_FORTIFY_SOURCE=1"); err != nil {
			t.Error(err)
		}

		cmd = ctx.must(callCompiler(ctx, ctx.cfg,
			ctx.newCommand(gccX86_64, "-D_FORTIFY_SOURCE=2", mainCc)))
		if err := verifyArgCount(cmd, 1, "-D_FORTIFY_SOURCE=2"); err != nil {
			t.Error(err)
		}
	})
}

func TestKeepSanitizerFlagsIfSanitizeGivenInCommonFlags(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		ctx.cfg.commonFlags = []string{"-fsanitize=kernel-address"}
		cmd := ctx.must(callCompiler(ctx, ctx.cfg,
			ctx.newCommand(gccX86_64, "-Wl,--no-undefined", mainCc)))
		if err := verifyArgCount(cmd, 1, "-Wl,--no-undefined"); err != nil {
			t.Error(err)
		}
	})
}
