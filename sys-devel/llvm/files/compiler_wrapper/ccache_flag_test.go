// Copyright 2019 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"path/filepath"
	"testing"
)

func TestCallCCacheGivenConfig(t *testing.T) {
	withCCacheEnabledTestContext(t, func(ctx *testContext) {
		cmd := ctx.must(callCompiler(ctx, ctx.cfg,
			ctx.newCommand(gccX86_64, mainCc)))
		if err := verifyPath(cmd, "/usr/bin/ccache"); err != nil {
			t.Error(err)
		}
		if err := verifyArgOrder(cmd, gccX86_64+".real", mainCc); err != nil {
			t.Error(err)
		}
	})
}

func TestNotCallCCacheGivenConfig(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		cmd := ctx.must(callCompiler(ctx, ctx.cfg,
			ctx.newCommand(gccX86_64, mainCc)))
		if err := verifyPath(cmd, gccX86_64+".real"); err != nil {
			t.Error(err)
		}
	})
}

func TestCallCCacheGivenEnviron(t *testing.T) {
	withCCacheEnabledTestContext(t, func(ctx *testContext) {
		ctx.env = append(ctx.env, "COMPILER_WRAPPER_FORCE_CCACHE=1")

		cmd := ctx.must(callCompiler(ctx, ctx.cfg,
			ctx.newCommand(gccX86_64, mainCc)))
		if err := verifyPath(cmd, "/usr/bin/ccache"); err != nil {
			t.Error(err)
		}
		if err := verifyArgOrder(cmd, gccX86_64+".real", mainCc); err != nil {
			t.Error(err)
		}
	})
}

func TestNotCallCCacheGivenEnviron(t *testing.T) {
	withCCacheEnabledTestContext(t, func(ctx *testContext) {
		ctx.env = append(ctx.env, "COMPILER_WRAPPER_FORCE_CCACHE=0")

		cmd := ctx.must(callCompiler(ctx, ctx.cfg,
			ctx.newCommand(gccX86_64, mainCc)))
		if err := verifyPath(cmd, gccX86_64+".real"); err != nil {
			t.Error(err)
		}
	})
}

func TestNotCallCCacheGivenConfigAndNoCCacheArg(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		cmd := ctx.must(callCompiler(ctx, ctx.cfg,
			ctx.newCommand(gccX86_64, "-noccache", mainCc)))
		if err := verifyPath(cmd, gccX86_64+".real"); err != nil {
			t.Error(err)
		}
		if err := verifyArgCount(cmd, 0, "-noccache"); err != nil {
			t.Error(err)
		}
	})
}

func TestSetCacheDir(t *testing.T) {
	withCCacheEnabledTestContext(t, func(ctx *testContext) {
		cmd := ctx.must(callCompiler(ctx, ctx.cfg,
			ctx.newCommand(gccX86_64, mainCc)))
		if err := verifyEnvUpdate(cmd, "CCACHE_DIR=/var/cache/distfiles/ccache"); err != nil {
			t.Error(err)
		}
	})
}

func TestSetCacheUmask(t *testing.T) {
	withCCacheEnabledTestContext(t, func(ctx *testContext) {
		cmd := ctx.must(callCompiler(ctx, ctx.cfg,
			ctx.newCommand(gccX86_64, mainCc)))
		if err := verifyEnvUpdate(cmd, "CCACHE_UMASK=002"); err != nil {
			t.Error(err)
		}
	})
}

func TestUpdateSandboxRewriteWithValue(t *testing.T) {
	withCCacheEnabledTestContext(t, func(ctx *testContext) {
		cmd := ctx.must(callCompiler(ctx, ctx.cfg,
			ctx.newCommand(gccX86_64, mainCc)))
		if err := verifyNoEnvUpdate(cmd, "SANDBOX_WRITE"); err != nil {
			t.Error(err)
		}

		ctx.env = []string{"SANDBOX_WRITE=xyz"}
		cmd = ctx.must(callCompiler(ctx, ctx.cfg,
			ctx.newCommand(gccX86_64, mainCc)))
		if err := verifyEnvUpdate(cmd,
			"SANDBOX_WRITE=xyz:/var/cache/distfiles/ccache"); err != nil {
			t.Error(err)
		}
	})
}

func TestUpdateSandboxRewriteWithoutValue(t *testing.T) {
	withCCacheEnabledTestContext(t, func(ctx *testContext) {
		cmd := ctx.must(callCompiler(ctx, ctx.cfg,
			ctx.newCommand(gccX86_64, mainCc)))
		if err := verifyNoEnvUpdate(cmd, "SANDBOX_WRITE"); err != nil {
			t.Error(err)
		}

		ctx.env = []string{"SANDBOX_WRITE="}
		cmd = ctx.must(callCompiler(ctx, ctx.cfg,
			ctx.newCommand(gccX86_64, mainCc)))
		if err := verifyEnvUpdate(cmd,
			"SANDBOX_WRITE=:/var/cache/distfiles/ccache"); err != nil {
			t.Error(err)
		}
	})
}

func TestClearCCacheDisableWithValue(t *testing.T) {
	withCCacheEnabledTestContext(t, func(ctx *testContext) {
		cmd := ctx.must(callCompiler(ctx, ctx.cfg,
			ctx.newCommand(gccX86_64, mainCc)))
		if err := verifyNoEnvUpdate(cmd, "CCACHE_DISABLE"); err != nil {
			t.Error(err)
		}

		ctx.env = []string{"CCACHE_DISABLE=true"}
		cmd = ctx.must(callCompiler(ctx, ctx.cfg,
			ctx.newCommand(gccX86_64, mainCc)))
		if err := verifyEnvUpdate(cmd, "CCACHE_DISABLE="); err != nil {
			t.Error(err)
		}
	})
}

func TestClearCCacheDisableWithoutValue(t *testing.T) {
	withCCacheEnabledTestContext(t, func(ctx *testContext) {
		cmd := ctx.must(callCompiler(ctx, ctx.cfg,
			ctx.newCommand(gccX86_64, mainCc)))
		if err := verifyNoEnvUpdate(cmd, "CCACHE_DISABLE"); err != nil {
			t.Error(err)
		}

		ctx.env = []string{"CCACHE_DISABLE="}
		cmd = ctx.must(callCompiler(ctx, ctx.cfg,
			ctx.newCommand(gccX86_64, mainCc)))
		if err := verifyEnvUpdate(cmd, "CCACHE_DISABLE="); err != nil {
			t.Error(err)
		}
	})
}

func TestAddCCacheCpp2FlagForClang(t *testing.T) {
	withCCacheEnabledTestContext(t, func(ctx *testContext) {
		cmd := ctx.must(callCompiler(ctx, ctx.cfg,
			ctx.newCommand(clangX86_64, mainCc)))
		if err := verifyEnvUpdate(cmd, "CCACHE_CPP2=yes"); err != nil {
			t.Error(err)
		}
	})
}

func TestOmitCCacheCpp2FlagForGcc(t *testing.T) {
	withCCacheEnabledTestContext(t, func(ctx *testContext) {
		cmd := ctx.must(callCompiler(ctx, ctx.cfg,
			ctx.newCommand(gccX86_64, mainCc)))
		if err := verifyNoEnvUpdate(cmd, "CCACHE_CPP2"); err != nil {
			t.Error(err)
		}
	})
}

func withCCacheEnabledTestContext(t *testing.T, work func(ctx *testContext)) {
	withTestContext(t, func(ctx *testContext) {
		ctx.cfg.useCCache = true
		work(ctx)
	})
}

func TestRusagePreventsCCache(t *testing.T) {
	withCCacheEnabledTestContext(t, func(ctx *testContext) {
		ctx.NoteTestWritesToUmask()

		ctx.env = append(ctx.env, "TOOLCHAIN_RUSAGE_OUTPUT="+filepath.Join(ctx.tempDir, "rusage.log"))
		cmd := ctx.must(callCompiler(ctx, ctx.cfg,
			ctx.newCommand(gccX86_64, mainCc)))
		if err := verifyPath(cmd, gccX86_64+".real"); err != nil {
			t.Error(err)
		}
	})
}

func TestCcacheIsDisabledInSrcConfigure(t *testing.T) {
	withCCacheEnabledTestContext(t, func(ctx *testContext) {
		ctx.NoteTestWritesToUmask()

		ctx.env = append(ctx.env, "EBUILD_PHASE=configure")
		cmd := ctx.must(callCompiler(ctx, ctx.cfg,
			ctx.newCommand(gccX86_64, mainCc)))
		if err := verifyPath(cmd, gccX86_64+".real"); err != nil {
			t.Error(err)
		}
	})
}
