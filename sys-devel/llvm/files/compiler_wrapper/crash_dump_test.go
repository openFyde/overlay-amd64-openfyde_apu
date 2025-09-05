// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"path/filepath"
	"testing"
)

func TestHardenedConfigDoesNotSpecifyCrashDirWhenNotInEnv(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		cmd := ctx.must(callCompiler(ctx, ctx.cfg, ctx.newCommand(clangX86_64, mainCc)))
		if err := verifyArgCount(cmd, 0, "-fcrash-diagnostics-dir=.*"); err != nil {
			t.Error(err)
		}
	})
}

func TestHardenedConfigSpecifiesCrashDirWhenInEnv(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		artifactsDir := ctx.setArbitraryClangArtifactsDir()
		crashDir := filepath.Join(artifactsDir, clangCrashArtifactsSubdir)

		cmd := ctx.must(callCompiler(ctx, ctx.cfg, ctx.newCommand(clangX86_64, mainCc)))
		if err := verifyArgCount(cmd, 1, "-fcrash-diagnostics-dir="+crashDir); err != nil {
			t.Error(err)
		}
	})
}

func TestHardenedConfigDoesNotSpecifyCrashDirForGCC(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		ctx.setArbitraryClangArtifactsDir()

		cmd := ctx.must(callCompiler(ctx, ctx.cfg, ctx.newCommand(gccX86_64, mainCc)))
		if err := verifyArgCount(cmd, 0, "-fcrash-diagnostics-dir=.*"); err != nil {
			t.Error(err)
		}
	})
}
