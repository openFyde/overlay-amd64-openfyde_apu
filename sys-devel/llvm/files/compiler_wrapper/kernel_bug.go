// Copyright 2021 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
package main

import (
	"bytes"
	"strings"
)

// crbug.com/1166017

// GCC will sometimes fail to wait on subprocesses due to this kernel bug. It always fails the
// compilation and prints "Unknown error 512" in that case.
func containsTracesOfKernelBug(buf []byte) bool {
	return bytes.Contains(buf, []byte("Unknown error 512"))
}

func errorContainsTracesOfKernelBug(err error) bool {
	// We'll get errors that look like "waitid: errno 512." Presumably, this isn't specific to
	// waitid, so just try to match the "errno 512" ending.
	return err != nil && strings.HasSuffix(err.Error(), "errno 512")
}
