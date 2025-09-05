// Copyright 2019 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"strings"
)

func processThumbCodeFlags(builder *commandBuilder) {
	arch := builder.target.arch
	if builder.target.abi != "eabi" && (strings.HasPrefix(arch, "armv7") || strings.HasPrefix(arch, "armv8")) {
		// ARM32 specfic:
		// 1. Generate thumb codes by default.  GCC is configured with
		//    --with-mode=thumb and defaults to thumb mode already.  This
		//    changes the default behavior of clang and doesn't affect GCC.
		// 2. Do not force frame pointers on ARM32 (https://crbug.com/693137).
		builder.addPreUserArgs("-mthumb")
		builder.transformArgs(func(arg builderArg) string {
			if !arg.fromUser && arg.value == "-fno-omit-frame-pointer" {
				return ""
			}
			return arg.value
		})
	}
}
