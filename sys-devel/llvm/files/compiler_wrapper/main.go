// Copyright 2019 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// This binary requires the following linker variables:
//   - main.UseCCache: Whether to use ccache.
//   - main.ConfigName: Name of the configuration to use.
//     See config.go for the supported values.
//
// The script ./build simplifies the call to `go build`.
// E.g. ./build --use_ccache=true --config=cros.hardened will build a
// binary that uses the ccache for ChromeOS with hardened flags.
//
// Test arguments:
//   - updategolden: To update the golden results for the wrapper. Without it,
//     the tests will verify that the wrapper output matches the goldens.
//   - rungolden: To filter the golden tests by a regex for the wrapper env, path and args.
//
// Examples:
//   - run all tests in isolation:
//     go test third_party/toolchain-utils/compiler_wrapper/ -v
package main

import (
	"log"
	"os"
)

func main() {
	env, err := newProcessEnv()
	if err != nil {
		log.Fatal(err)
	}
	cfg, err := getRealConfig()
	if err != nil {
		log.Fatal(err)
	}
	// Note: callCompiler will exec the command. Only in case of
	// an error or when we run other commands like bisect
	// will this os.Exit be called.
	os.Exit(callCompiler(env, cfg, newProcessCommand()))
}
