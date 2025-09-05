#!/usr/bin/env python3
# Copyright 2019 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

"""Build script that builds a binary from a bundle."""

import argparse
import os.path
import re
import subprocess
import sys


def parse_args():
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "--config",
        required=True,
        choices=["cros.hardened", "cros.nonhardened", "cros.host", "android"],
    )
    parser.add_argument(
        "--use_ccache", required=True, choices=["true", "false"]
    )
    parser.add_argument(
        "--use_llvm_next", default="false", choices=["true", "false"]
    )
    parser.add_argument("--output_file", required=True, type=str)
    parser.add_argument(
        "--static",
        choices=["true", "false"],
        help="If true, produce a static wrapper. Autodetects a good value if "
        "unspecified.",
    )

    version_args = parser.add_mutually_exclusive_group()
    version_args.add_argument(
        "--version",
        help="""
        A string to pass to `go` that instructs the compiler wrapper about what
        version to print. Automatically selects the current git commit SHA if
        this is left unspecified.
        """,
    )
    parser.add_argument(
        "--version_suffix",
        help="""
        A string appended to the **computed** version of the wrapper. This is
        appended directly without any delimiter. Incompatible with
        `--version`.
        """,
    )
    parser.add_argument(
        "--llvm_revision",
        type=int,
        help="""
        The synthetic revision of LLVM, as an integer. If not specified, none
        will be given to the compiler wrapper (see `LlvmRevision` in the wrapper
        for the implications of this).
        """,
    )
    args = parser.parse_args()

    if args.static is None:
        args.static = "cros" not in args.config
    else:
        args.static = args.static == "true"

    return args


def calc_go_args(args, version, build_dir, output_file):
    # These seem unnecessary, and might lead to breakages with Go's ldflag
    # parsing. Don't allow them.
    if "'" in version:
        raise ValueError("`version` should not contain single quotes")

    ldFlags = [
        "-X",
        "main.ConfigName=" + args.config,
        "-X",
        "main.UseCCache=" + args.use_ccache,
        "-X",
        "main.UseLlvmNext=" + args.use_llvm_next,
        "-X",
        # Quote this, as `version` may have spaces in it.
        "'main.Version=" + version + "'",
    ]

    if args.llvm_revision is not None:
        ldFlags += ("-X", f"main.LlvmRevision={args.llvm_revision}")

    # If the wrapper is intended for ChromeOS, we need to use libc's exec.
    extra_args = []
    if not args.static:
        extra_args += ["-tags", "libc_exec"]

    if args.config == "android":
        # If android_llvm_next_flags.go DNE, we'll get an obscure "no
        # llvmNextFlags" build error; complaining here is clearer.
        if not os.path.exists(
            os.path.join(build_dir, "android_llvm_next_flags.go")
        ):
            sys.exit(
                "In order to build the Android wrapper, you must have a local "
                "android_llvm_next_flags.go file; please see "
                "cros_llvm_next_flags.go."
            )
        extra_args += ["-tags", "android_llvm_next_flags"]

    return [
        "go",
        "build",
        "-o",
        output_file,
        "-ldflags",
        " ".join(ldFlags),
    ] + extra_args


def read_version(build_dir):
    version_path = os.path.join(build_dir, "VERSION")
    if os.path.exists(version_path):
        with open(version_path, "r", encoding="utf-8") as r:
            return r.read()

    last_commit_msg = subprocess.check_output(
        ["git", "-C", build_dir, "log", "-1", "--pretty=%B"], encoding="utf-8"
    )
    # Use last found change id to support reverts as well.
    change_ids = re.findall(r"Change-Id: (\w+)", last_commit_msg)
    if not change_ids:
        sys.exit("Couldn't find Change-Id in last commit message.")
    return change_ids[-1]


def main():
    args = parse_args()
    build_dir = os.path.dirname(__file__)

    if args.version:
        version = args.version
    else:
        version = read_version(build_dir)
        if args.version_suffix:
            version += args.version_suffix

    # Note: Go does not support using absolute package names.
    # So we run go inside the directory of the the build file.
    output_file = os.path.abspath(args.output_file)
    subprocess.check_call(
        calc_go_args(args, version, build_dir, output_file), cwd=build_dir
    )

    # b/203821449: we're occasionally seeing very small (and non-functional)
    # compiler-wrapper binaries on SDK builds. To help narrow down why, add a
    # size check here. Locally, the wrapper is 1.9MB, so warning on <1MB
    # shouldn't flag false-positives.
    size = os.path.getsize(output_file)
    min_size_bytes = 1024 * 1024
    if size < min_size_bytes:
        raise ValueError(
            f"Compiler wrapper is {size:,} bytes; expected at "
            f"least {min_size_bytes:,}"
        )


if __name__ == "__main__":
    main()
