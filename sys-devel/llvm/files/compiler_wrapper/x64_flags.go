// Copyright 2019 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"strings"
)

func processX86Flags(builder *commandBuilder) {
	arch := builder.target.arch
	if strings.HasPrefix(arch, "x86_64") || startswithI86(arch) {
		builder.addPostUserArgs("-mno-movbe")
	}
}

// Returns true if s starts with i.86.
func startswithI86(s string) bool {
	return len(s) >= 4 && s[0] == 'i' && s[2:4] == "86"
}
