// Copyright 2019 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"fmt"
	"regexp"
	"testing"
)

func TestRemovePrintCmdlineArg(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		cmd := ctx.must(callCompiler(ctx, ctx.cfg, ctx.newCommand(gccX86_64, "-print-cmdline", mainCc)))
		if err := verifyArgCount(cmd, 0, "-print-cmdline"); err != nil {
			t.Error(err)
		}
	})
}

func TestPrintCompilerCommand(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		ctx.must(callCompiler(ctx, ctx.cfg, ctx.newCommand(gccX86_64, "-print-cmdline", mainCc)))
		if matched, _ := regexp.MatchString(`cd '.*' && '.*/x86_64-cros-linux-gnu-gcc.real'.*'main.cc'`, ctx.stderrString()); !matched {
			t.Errorf("sub command not printed to stderr. Got: %s", ctx.stderrString())
		}
	})
}

func TestPrintNestedCommand(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		// Note: -clang-syntax calls clang to check the syntax
		ctx.must(callCompiler(ctx, ctx.cfg, ctx.newCommand(gccX86_64, "-print-cmdline", "-clang-syntax", mainCc)))
		if matched, _ := regexp.MatchString(`cd '.*' && '.*usr/bin/clang'.*'main.cc'.*'-fsyntax-only'`, ctx.stderrString()); !matched {
			t.Errorf("sub command not printed to stderr. Got: %s", ctx.stderrString())
		}
	})
}

func TestPrintCmdWd(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		printCmd(ctx, &command{
			Path: "/somepath",
		})
		if ctx.stderrString() != fmt.Sprintf("cd '%s' && '/somepath'\n", ctx.tempDir) {
			t.Errorf("unexpected result. Got: %s", ctx.stderrString())
		}
	})
}

func TestPrintCmdAbsolutePath(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		printCmd(ctx, &command{
			Path: "somepath",
		})
		if ctx.stderrString() != fmt.Sprintf("cd '%s' && '%s/somepath'\n", ctx.tempDir, ctx.tempDir) {
			t.Errorf("unexpected result. Got: %s", ctx.stderrString())
		}
	})
}

func TestPrintCmdEnvUpdates(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		printCmd(ctx, &command{
			Path:       "/somepath",
			EnvUpdates: []string{"a=b"},
		})
		if ctx.stderrString() != fmt.Sprintf("cd '%s' && env 'a=b' '/somepath'\n", ctx.tempDir) {
			t.Errorf("unexpected result. Got: %s", ctx.stderrString())
		}
	})
}

func TestPrintCmdArgs(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		printCmd(ctx, &command{
			Path: "/somepath",
			Args: []string{"-a"},
		})
		if ctx.stderrString() != fmt.Sprintf("cd '%s' && '/somepath' '-a'\n", ctx.tempDir) {
			t.Errorf("unexpected result. Got: %s", ctx.stderrString())
		}
	})
}
