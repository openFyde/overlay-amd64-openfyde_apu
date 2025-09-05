// Copyright 2019 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

func processPrintCmdlineFlag(builder *commandBuilder) {
	printCmd := false
	builder.transformArgs(func(arg builderArg) string {
		if arg.value == "-print-cmdline" {
			printCmd = true
			return ""
		}
		return arg.value
	})
	if printCmd {
		builder.env = &printingEnv{builder.env}
	}
}
