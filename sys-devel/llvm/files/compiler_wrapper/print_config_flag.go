// Copyright 2019 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import "fmt"

func processPrintConfigFlag(builder *commandBuilder) {
	printConfig := false
	builder.transformArgs(func(arg builderArg) string {
		if arg.value == "-print-config" {
			printConfig = true
			return ""
		}
		return arg.value
	})
	if printConfig {
		fmt.Fprintf(builder.env.stderr(), "wrapper config: %#v\n", *builder.cfg)
	}
}
