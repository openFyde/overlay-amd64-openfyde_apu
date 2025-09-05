// Copyright 2019 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"strings"
	"testing"
)

func TestRemovePrintConfigArg(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		cmd := ctx.must(callCompiler(ctx, ctx.cfg, ctx.newCommand(gccX86_64, "-print-config", mainCc)))
		if err := verifyArgCount(cmd, 0, "-print-config"); err != nil {
			t.Error(err)
		}
	})
}

func TestPrintConfig(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		ctx.must(callCompiler(ctx, ctx.cfg, ctx.newCommand(gccX86_64, "-print-config", mainCc)))
		if !strings.Contains(ctx.stderrString(), "wrapper config: main.config{") {
			t.Errorf("config not printed to stderr. Got: %s", ctx.stderrString())
		}
	})
}
