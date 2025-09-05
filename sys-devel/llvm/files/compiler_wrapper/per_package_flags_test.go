// Copyright 2025 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import "testing"

func TestTryGetLLVMRevisionWorks(t *testing.T) {
	initialLlvmRevision := LlvmRevision
	defer func() { LlvmRevision = initialLlvmRevision }()

	LlvmRevision = ""
	if _, ok, err := tryGetLlvmRevision(); err != nil {
		t.Errorf("Blank llvm revision returned an error unexpectedly")
	} else if ok {
		t.Errorf("Blank llvm revision returned ok==true unexpectedly")
	}

	LlvmRevision = "123"
	if rev, ok, err := tryGetLlvmRevision(); err != nil {
		t.Errorf("123 llvm revision returned an error unexpectedly")
	} else if !ok {
		t.Errorf("123 llvm revision returned ok==false unexpectedly")
	} else if rev != 123 {
		t.Errorf("123 llvm revision returned rev==%v unexpectedly", rev)
	}

	LlvmRevision = "not-a-number"
	if _, _, err := tryGetLlvmRevision(); err == nil {
		t.Errorf("invalid llvm revision number didn't return an error")
	}
}
