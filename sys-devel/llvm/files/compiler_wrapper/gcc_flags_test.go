// Copyright 2019 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"testing"
)

func TestCallRealGcc(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		cmd := ctx.must(callCompiler(ctx, ctx.cfg,
			ctx.newCommand(gccX86_64, mainCc)))
		if err := verifyPath(cmd, gccX86_64+".real"); err != nil {
			t.Error(err)
		}
	})
}

func TestCallRealGccForOtherNames(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		cmd := ctx.must(callCompiler(ctx, ctx.cfg,
			ctx.newCommand("./x86_64-cros-linux-gnu-somename", mainCc)))
		if err := verifyPath(cmd, "\\./x86_64-cros-linux-gnu-somename.real"); err != nil {
			t.Error(err)
		}
	})
}

func TestConvertClangToGccFlags(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		var tests = []struct {
			in  string
			out string
		}{
			{"-march=alderlake", "-march=skylake"},
		}

		for _, tt := range tests {
			cmd := ctx.must(callCompiler(ctx, ctx.cfg,
				ctx.newCommand(gccX86_64, tt.in, mainCc)))
			if err := verifyArgCount(cmd, 0, tt.in); err != nil {
				t.Error(err)
			}
			if err := verifyArgOrder(cmd, tt.out, mainCc); err != nil {
				t.Error(err)
			}
		}
	})
}

func TestFilterUnsupportedGccFlags(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		cmd := ctx.must(callCompiler(ctx, ctx.cfg,
			ctx.newCommand(gccX86_64, "-Xcompiler", mainCc)))
		if err := verifyArgCount(cmd, 0, "-Xcompiler"); err != nil {
			t.Error(err)
		}
	})
}
