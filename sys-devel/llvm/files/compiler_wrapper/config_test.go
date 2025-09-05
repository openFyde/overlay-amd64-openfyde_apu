// Copyright 2019 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"testing"
)

func TestRealConfigWithUseCCacheFlag(t *testing.T) {
	resetGlobals()
	defer resetGlobals()
	ConfigName = "cros.hardened"
	UseLlvmNext = "false"

	UseCCache = "false"
	cfg, err := getRealConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.useCCache {
		t.Fatal("UseCCache: Expected false got true")
	}

	UseCCache = "true"
	cfg, err = getRealConfig()
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.useCCache {
		t.Fatal("UseCCache: Expected true got false")
	}

	UseCCache = "invalid"
	if _, err := getRealConfig(); err == nil {
		t.Fatalf("UseCCache: Expected an error, got none")
	}
}

func TestRealConfigWithUseLLvmFlag(t *testing.T) {
	resetGlobals()
	defer resetGlobals()
	ConfigName = "cros.hardened"
	UseCCache = "false"

	UseLlvmNext = "false"
	cfg, err := getRealConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.useLlvmNext {
		t.Fatal("UseLLvmNext: Expected not to be used")
	}

	UseLlvmNext = "true"
	cfg, err = getRealConfig()
	if err != nil {
		t.Fatal(err)
	}

	if !cfg.useLlvmNext {
		t.Fatal("UseLLvmNext: Expected to be used")
	}

	UseLlvmNext = "invalid"
	if _, err := getRealConfig(); err == nil {
		t.Fatalf("UseLlvmNext: Expected an error, got none")
	}
}

func TestRealConfigWithConfigNameFlag(t *testing.T) {
	resetGlobals()
	defer resetGlobals()
	UseCCache = "false"
	UseLlvmNext = "false"

	ConfigName = "cros.hardened"
	cfg, err := getRealConfig()
	if err != nil {
		t.Fatal(err)
	}
	if !isSysrootHardened(cfg) || cfg.isHostWrapper {
		t.Fatalf("ConfigName: Expected sysroot hardened config. Got: %#v", cfg)
	}

	ConfigName = "cros.nonhardened"
	cfg, err = getRealConfig()
	if err != nil {
		t.Fatal(err)
	}
	if isSysrootHardened(cfg) || cfg.isHostWrapper {
		t.Fatalf("ConfigName: Expected sysroot non hardened config. Got: %#v", cfg)
	}

	ConfigName = "cros.host"
	cfg, err = getRealConfig()
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.isHostWrapper {
		t.Fatalf("ConfigName: Expected clang host config. Got: %#v", cfg)
	}

	ConfigName = "android"
	cfg, err = getRealConfig()
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.isAndroidWrapper {
		t.Fatalf("ConfigName: Expected clang host config. Got: %#v", cfg)
	}

	ConfigName = "invalid"
	if _, err := getRealConfig(); err == nil {
		t.Fatalf("ConfigName: Expected an error, got none")
	}
}

func isSysrootHardened(cfg *config) bool {
	for _, arg := range cfg.commonFlags {
		if arg == "-D_FORTIFY_SOURCE=3" {
			return true
		}
	}
	return false
}

func resetGlobals() {
	// Set all global variables to a defined state.
	UseLlvmNext = "unknown"
	ConfigName = "unknown"
	UseCCache = "unknown"
}
