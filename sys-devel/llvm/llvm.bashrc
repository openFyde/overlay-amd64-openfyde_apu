# Copyright 2019 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

# llvm failed to build with "-fsanitize=vptr", https://crbug.com/943766.
cros_pre_src_prepare_filter_sanitizers() {
	tc-is-clang && append-flags "-fno-sanitize=vptr"
}
