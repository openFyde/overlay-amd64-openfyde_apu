// Copyright 2025 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"fmt"
	"strconv"
	"strings"
)

const crostcApplyFlagsForEnv = "CROSTC_ADD_IMPLICIT_CFLAGS_FOR"
const crostcApplyFlagsForFlag = "-D_CROSTC_ADD_IMPLICIT_CFLAGS_FOR="

func getExtraPerPackageFlags(llvmRev int, packageName string) []string {
	return nil
}

// Tries to get the LLVM revision supplied to the wrapper's build. Returns an error if there's
// a parse error; otherwise, returns the given revision, and whether a revision was specified.
func tryGetLlvmRevision() (int, bool, error) {
	if LlvmRevision == "" {
		return 0, false, nil
	}
	n, err := strconv.ParseInt(LlvmRevision, 10, 0)
	return int(n), true, err
}

func processPerPackageFlags(cfg *config, builder *commandBuilder) error {
	if cfg.isAndroidWrapper {
		return nil
	}

	packageName := ""
	builder.transformArgs(func(arg builderArg) string {
		if pkg := strings.TrimPrefix(arg.value, crostcApplyFlagsForFlag); len(pkg) != len(arg.value) {
			packageName = pkg
		}
		// Leave the -D as part of the command-line; it shouldn't hurt, and leads to one fewer
		// divergence between this wrapper and the actual Clang invocation.
		return arg.value
	})

	if packageName == "" {
		val, _ := builder.env.getenv(crostcApplyFlagsForEnv)
		if val == "" {
			return nil
		}
		packageName = val
	}

	llvmRev, ok, err := tryGetLlvmRevision()
	if err != nil {
		return fmt.Errorf("parsing provided LLVM revision: %v", err)
	}
	if ok {
		builder.addPostUserArgs(getExtraPerPackageFlags(llvmRev, packageName)...)
	}
	return nil
}
