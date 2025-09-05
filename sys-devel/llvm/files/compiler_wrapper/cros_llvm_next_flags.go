// Copyright 2020 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

//go:build !android_llvm_next_flags
// +build !android_llvm_next_flags

package main

// This file defines extra flags for llvm-next testing for ChromeOS. Importantly, these flags don't
// apply to Android's llvm-next wrapper. Android's toolchain-utils copy has a
// `android_llvm_next_flags.go` file downstream that defines its llvm-next arguments. As you can
// probably infer, `android_llvm_next_flags.go` is only compiled if the `android_llvm_next_flags`
// tag is set.

// TODO: Enable test in config_test.go, once we have new llvm-next flags.
var llvmNextFlags = []string{}

var llvmNextPostFlags = []string{}
