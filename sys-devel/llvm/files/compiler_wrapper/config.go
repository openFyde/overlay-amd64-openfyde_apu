// Copyright 2019 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"strconv"
)

type config struct {
	// TODO: Refactor this flag into more generic configuration properties.
	isHostWrapper    bool
	isAndroidWrapper bool
	// Whether to use ccache.
	useCCache bool
	// Whether llvmNext wrapper.
	useLlvmNext bool
	// Flags to add to gcc and clang.
	commonFlags []string
	// Flags to add to gcc only.
	gccFlags []string
	// Flags to add to clang only.
	clangFlags []string
	// Flags to add to clang only, AFTER user flags (cannot be overridden
	// by the user).
	clangPostFlags []string
	// Toolchain root path relative to the wrapper binary.
	clangRootRelPath string
	gccRootRelPath   string
	// Version. Only exposed via -print-config.
	version string
}

// Version can be set via a linker flag.
// Values fills config.version.
var Version = ""

// UseCCache can be set via a linker flag.
// Value will be passed to strconv.ParseBool.
// E.g. go build -ldflags '-X main.UseCCache=true'.
var UseCCache = "unknown"

// UseLlvmNext can be set via a linker flag.
// Value will be passed to strconv.ParseBool.
// E.g. go build -ldflags '-X main.UseLlvmNext=true'.
var UseLlvmNext = "unknown"

// ConfigName can be set via a linker flag.
// Value has to be one of:
// - "cros.hardened"
// - "cros.nonhardened"
var ConfigName = "unknown"

// LlvmRevision can be set via a linker flag.
// Value, if provided, will be passed to strconv.ParseInt. If none is provided, logic
// predicated on the current LLVM revision will be skipped.
// E.g. go build -ldflags '-X main.LlvmRevision=12345'.
var LlvmRevision = ""

// Returns the configuration matching the UseCCache and ConfigName.
func getRealConfig() (*config, error) {
	useCCache, err := strconv.ParseBool(UseCCache)
	if err != nil {
		return nil, wrapErrorwithSourceLocf(err, "invalid format for UseCCache")
	}
	useLlvmNext, err := strconv.ParseBool(UseLlvmNext)
	if err != nil {
		return nil, wrapErrorwithSourceLocf(err, "invalid format for UseLLvmNext")
	}
	config, err := getConfig(ConfigName, useCCache, useLlvmNext, Version)
	if err != nil {
		return nil, err
	}
	return config, nil
}

func isAndroidConfig() bool {
	return ConfigName == "android"
}

func getConfig(configName string, useCCache bool, useLlvmNext bool, version string) (*config, error) {
	cfg := (*config)(nil)
	switch configName {
	case "cros.hardened":
		cfg = getCrosHardenedConfig()
	case "cros.nonhardened":
		cfg = getCrosNonHardenedConfig()
	case "cros.host":
		cfg = getCrosHostConfig()
	case "android":
		cfg = getAndroidConfig()
	default:
		return nil, newErrorwithSourceLocf("unknown config name: %s", configName)
	}
	cfg.useCCache = useCCache
	cfg.useLlvmNext = useLlvmNext
	if useLlvmNext {
		cfg.clangFlags = append(cfg.clangFlags, llvmNextFlags...)
		cfg.clangPostFlags = append(cfg.clangPostFlags, llvmNextPostFlags...)
	}
	cfg.version = version
	return cfg, nil
}

func crosCommonClangFlags() []string {
	// Temporarily disable tautological-*-compare chromium:778316.
	// Temporarily add no-unknown-warning-option to deal with old clang versions.
	// Temporarily disable Wdeprecated-declarations. b/193860318
	// b/230345382: Temporarily disable Wimplicit-function-declaration.
	// b/231987783: Temporarily disable Wimplicit-int.
	return []string{
		"-Qunused-arguments",
		"-Werror=poison-system-directories",
		"-Wno-deprecated-declarations",
		"-Wno-error=implicit-function-declaration",
		"-Wno-error=implicit-int",
		"-Wno-unknown-warning-option",
		"-fdebug-default-version=5",
		"-Wno-int-conversion",
		"-Wno-incompatible-function-pointer-types",
		// TODO(b/316021385): Temporarily disables warnings for variable length arrays.
		"-Wno-error=vla-cxx-extension",
		"-D_LIBCPP_ENABLE_CXX17_REMOVED_FEATURES",
	}
}

func crosCommonClangPostFlags() []string {
	// Flags added to the _end_ of every build command. If a flag is added here, file a bug at
	// go/crostc-bug to clean it up. Use of postflags is discouraged, since it prevents users
	// from determining their own preferences for warnings/etc.
	return []string{}
}

// Full hardening.
func getCrosHardenedConfig() *config {
	// Temporarily disable function splitting because of chromium:434751.
	return &config{
		clangRootRelPath: "../..",
		gccRootRelPath:   "../../../../..",
		// Pass "-fcommon" till the packages are fixed to work with new clang/gcc
		// default of "-fno-common", crbug.com/1060413.
		commonFlags: []string{
			"-fcommon",
			"-fstack-protector-strong",
			"-D_FORTIFY_SOURCE=3",
			"-fno-omit-frame-pointer",
		},
		gccFlags: []string{
			"-fno-reorder-blocks-and-partition",
			"-Wno-unused-local-typedefs",
			"-Wno-maybe-uninitialized",
		},
		// Temporarily disable Wsection since kernel gets a bunch of these. chromium:778867
		// crbug.com/1103065: -grecord-gcc-switches pollutes the Goma cache;
		//   removed that flag for now.
		clangFlags: append(
			crosCommonClangFlags(),
			"--unwindlib=libunwind",
			"-Wno-section",
			"-ftrivial-auto-var-init=zero",
		),
		clangPostFlags: crosCommonClangPostFlags(),
	}
}

// Flags to be added to non-hardened toolchain.
func getCrosNonHardenedConfig() *config {
	return &config{
		clangRootRelPath: "../..",
		gccRootRelPath:   "../../../../..",
		commonFlags:      []string{},
		gccFlags: []string{
			"-Wno-maybe-uninitialized",
			"-Wno-unused-local-typedefs",
			"-Wno-deprecated-declarations",
			"-Wtrampolines",
		},
		// Temporarily disable Wsection since kernel gets a bunch of these. chromium:778867
		clangFlags: append(
			crosCommonClangFlags(),
			"-Wno-section",
		),
		clangPostFlags: crosCommonClangPostFlags(),
	}
}

// Flags to be added to host toolchain.
func getCrosHostConfig() *config {
	return &config{
		isHostWrapper:    true,
		clangRootRelPath: "../..",
		gccRootRelPath:   "../..",
		// Pass "-fcommon" till the packages are fixed to work with new clang/gcc
		// default of "-fno-common", crbug.com/1060413.
		commonFlags: []string{
			"-fcommon",
		},
		gccFlags: []string{
			"-Wno-maybe-uninitialized",
			"-Wno-unused-local-typedefs",
			"-Wno-deprecated-declarations",
		},
		// crbug.com/1103065: -grecord-gcc-switches pollutes the Goma cache;
		//   removed that flag for now.
		clangFlags: append(
			crosCommonClangFlags(),
			"-Wno-unused-local-typedefs",
		),
		// Temporarily disable Wdeprecated-copy. b/191479033
		clangPostFlags: crosCommonClangPostFlags(),
	}
}

func getAndroidConfig() *config {
	return &config{
		isHostWrapper:    false,
		isAndroidWrapper: true,
		gccRootRelPath:   "./",
		clangRootRelPath: "./",
		commonFlags:      []string{},
		gccFlags:         []string{},
		clangFlags:       []string{},
		clangPostFlags:   []string{},
	}
}
