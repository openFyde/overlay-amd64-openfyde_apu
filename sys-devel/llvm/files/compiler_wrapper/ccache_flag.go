// Copyright 2019 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

func isInConfigureStage(env env) bool {
	val, present := env.getenv("EBUILD_PHASE")
	return present && val == "configure"
}

func processCCacheFlag(builder *commandBuilder) {
	// We should be able to share the objects across compilers as
	// the pre-processed output will differ.  This allows boards
	// that share compiler flags (like x86 boards) to share caches.
	const ccacheDir = "/var/cache/distfiles/ccache"

	useCCache := builder.cfg.useCCache
	builder.transformArgs(func(arg builderArg) string {
		if arg.value == "-noccache" {
			useCCache = false
			return ""
		}
		return arg.value
	})

	if force, present := builder.env.getenv("COMPILER_WRAPPER_FORCE_CCACHE"); present {
		switch force {
		case "0":
			useCCache = false
		case "1":
			useCCache = true
		}
	}

	// Disable ccache during portage's src_configure phase. Using ccache here is generally a
	// waste of time, since these files are very small. Experimentally, this speeds up
	// configuring by ~13%.
	if isInConfigureStage(builder.env) {
		useCCache = false
	}

	if useCCache {
		// Note: we used to also set CCACHE_BASEDIR but don't do it
		// anymore for reasons outlined in crrev.com/c/2103170.
		if _, present := builder.env.getenv("CCACHE_DISABLE"); present {
			// Portage likes to set this for us when it has FEATURES=-ccache.
			// The other vars we need to setup manually because of tools like
			// scons that scrubs the env before we get executed.
			builder.updateEnv("CCACHE_DISABLE=")
		}
		// If RESTRICT=sandbox is enabled, then sandbox won't be setup,
		// and the env vars won't be available for appending.
		if sandboxRewrite, present := builder.env.getenv("SANDBOX_WRITE"); present {
			builder.updateEnv("SANDBOX_WRITE=" + sandboxRewrite + ":" + ccacheDir)
		}

		// Make sure we keep the cached files group writable.
		builder.updateEnv("CCACHE_DIR="+ccacheDir, "CCACHE_UMASK=002")

		// ccache may generate false positive warnings.
		// Workaround bug https://crbug.com/649740
		if builder.target.compilerType == clangType {
			builder.updateEnv("CCACHE_CPP2=yes")
		}

		builder.wrapPath("/usr/bin/ccache")
	}
}
