// Copyright 2021 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"strings"
)

// Add "-static-libgcc" flag to compiler command line unless
// already specified by user or user has passed "-shared-libgcc".
func processLibGCCFlags(builder *commandBuilder) {
	fromUser := false
	for _, arg := range builder.args {
		if arg.fromUser && (strings.HasPrefix(arg.value, "-shared-libgcc") ||
			strings.HasPrefix(arg.value, "-static-libgcc")) {
			fromUser = true
			break
		}
	}
	if !fromUser {
		builder.addPreUserArgs("-static-libgcc")
	}
}
