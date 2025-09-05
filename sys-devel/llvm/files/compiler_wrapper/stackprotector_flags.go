// Copyright 2019 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

func processStackProtectorFlags(builder *commandBuilder) {
	fstackMap := map[string]bool{"-D__KERNEL__": true, "-fno-stack-protector": true, "-nodefaultlibs": true,
		"-nostdlib": true}

	fstack := false
	if builder.target.abi != "eabi" {
		for _, arg := range builder.args {
			if arg.fromUser && fstackMap[arg.value] {
				fstack = true
				break
			}
		}
	}
	if fstack {
		builder.addPreUserArgs("-fno-stack-protector")
		builder.transformArgs(func(arg builderArg) string {
			if !arg.fromUser && arg.value == "-fstack-protector-strong" {
				return ""
			}
			return arg.value
		})
	}
}
