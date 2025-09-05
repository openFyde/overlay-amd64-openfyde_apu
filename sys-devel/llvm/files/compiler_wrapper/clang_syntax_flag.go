// Copyright 2019 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

func processClangSyntaxFlag(builder *commandBuilder) (clangSyntax bool) {
	builder.transformArgs(func(arg builderArg) string {
		if arg.value == "-clang-syntax" {
			clangSyntax = true
			return ""
		}
		return arg.value
	})
	return clangSyntax
}

func checkClangSyntax(env env, clangCmd *command, gccCmd *command) (exitCode int, err error) {
	clangSyntaxCmd := &command{
		Path:       clangCmd.Path,
		Args:       append(clangCmd.Args, "-fsyntax-only", "-stdlib=libstdc++"),
		EnvUpdates: clangCmd.EnvUpdates,
	}

	getStdin, err := prebufferStdinIfNeeded(env, clangCmd)
	if err != nil {
		return 0, wrapErrorwithSourceLocf(err, "prebuffering stdin: %v", err)
	}

	exitCode, err = wrapSubprocessErrorWithSourceLoc(clangSyntaxCmd,
		env.run(clangSyntaxCmd, getStdin(), env.stdout(), env.stderr()))
	if err != nil || exitCode != 0 {
		return exitCode, err
	}
	return wrapSubprocessErrorWithSourceLoc(gccCmd,
		env.run(gccCmd, getStdin(), env.stdout(), env.stderr()))
}
