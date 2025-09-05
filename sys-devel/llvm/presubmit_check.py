#!/usr/bin/env python3
# Copyright 2023 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Presubmit checks for sys-devel/llvm."""

import os
import subprocess
import sys


def main():
    presubmit_files_str = os.environ.get("PRESUBMIT_FILES")
    if presubmit_files_str is None:
        sys.exit("Need a value for $PRESUBMIT_FILES")

    compiler_wrapper = "sys-devel/llvm/files/compiler_wrapper"

    # For now, just run `go test` if any files in `compiler_wrapper/` are
    # changed. PRESUBMIT_FILES is a line-delimited list of files to inspect.
    # N.B., if you add to this script, please be sure we still exit early if a
    # CL contains only files outside of sys-devel/llvm: this script is run on
    # every `repo upload` in `chromiumos-overlay`.
    if not any(
        x.startswith(compiler_wrapper) for x in presubmit_files_str.splitlines()
    ):
        return

    # Strictly speaking, these tests should be run consistently inside of a
    # chroot. Realistically, these tests are very portable and should ideally
    # be kept that way. If any weird issues happen inside of a chroot due to Go
    # versioning quirks, that'll be caught during `emerge llvm` in the CQ.
    return_code = subprocess.run(
        ["go", "test"],
        check=False,
        stdin=subprocess.DEVNULL,
        cwd=compiler_wrapper,
    ).returncode
    if return_code:
        sys.exit("compiler_wrapper tests failed")


if __name__ == "__main__":
    main()
