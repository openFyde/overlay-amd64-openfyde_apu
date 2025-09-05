// Copyright 2022 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"bytes"
	"fmt"
	"io"
	"regexp"
)

// ** HEY YOU, PERSON READING THIS! **
//
// Are you a dev who wants to make this work locally? Awesome! Please note that this **only** works
// for Clang. If that's OK, here's a checklist for you:
// [ ] Set `shouldUseCrashBuildsHeuristic = true` below.
// [ ] If you want this heuristic to operate during `src_configure` (rare), also set
// `allowAutoCrashInConfigure` to true.
// [ ] Modify `shouldAutocrashPostExec` to return `true` when the compiler's output/flags match what
// you want to crash on, and `false` otherwise.
// [ ] Run `./install_compiler_wrapper.sh` to install the updated wrapper.
// [ ] Run whatever command reproduces the error.
//
// If you need to make changes to your heuristic, repeat the above steps starting at
// `./install_compiler_wrapper.sh` until things seem to do what you want.
const (
	// Set this to true to use autocrashing logic.
	shouldUseCrashBuildsHeuristic = false
	// Set this to true to allow `shouldAutocrashPostExec` to check+crash configure steps.
	allowAutoCrashInConfigure = false
)

// shouldAutocrashPostExec returns true if we should automatically crash the compiler. This is
// called after the compiler is run. If it returns true, we'll re-execute the compiler with the bit
// of extra code necessary to crash it.
func shouldAutocrashPostExec(env env, cfg *config, originalCmd *command, runInfo compilerExecInfo) bool {
	// ** TODO, DEAR READER: ** Fill this in. Below are a few `if false {` blocks that should
	// work for common use-cases. You're encouraged to change them to `if true {` if they suit
	// your needs.

	// Return true if `error: some error message` is contained in the run's stderr.
	if false {
		return bytes.Contains(runInfo.stderr, []byte("error: some error message"))
	}

	// Return true if `foo.c:${line_number}: error: some error message` appears in the run's
	// stderr. Otherwise, return false.
	if false {
		r := regexp.MustCompile(`foo\.c:\d+: error: some error message`)
		return r.Match(runInfo.stderr)
	}

	// Return true if there's a `-fjust-give-up` flag in the compiler's invocation.
	if false {
		for _, flag := range originalCmd.Args {
			if flag == "-fjust-give-up" {
				return true
			}
		}

		return false
	}

	panic("Please fill in `shouldAutocrashPostExec` with meaningful logic.")
}

type compilerExecInfo struct {
	exitCode       int
	stdout, stderr []byte
}

// ** Below here are implementation details. If all you want is autocrashing behavior, you don't
// need to keep reading. **
const (
	autocrashProgramLine = "\n#pragma clang __debug parser_crash"
)

type buildWithAutocrashPredicates struct {
	allowInConfigure bool
	shouldAutocrash  func(env, *config, *command, compilerExecInfo) bool
}

func buildWithAutocrash(env env, cfg *config, originalCmd *command) (exitCode int, err error) {
	return buildWithAutocrashImpl(env, cfg, originalCmd, buildWithAutocrashPredicates{
		allowInConfigure: allowAutoCrashInConfigure,
		shouldAutocrash:  shouldAutocrashPostExec,
	})
}

func buildWithAutocrashImpl(env env, cfg *config, originalCmd *command, preds buildWithAutocrashPredicates) (exitCode int, err error) {
	stdinBuffer := (*bytes.Buffer)(nil)
	subprocStdin := io.Reader(nil)
	invocationUsesStdinAsAFile := needStdinTee(originalCmd)
	if invocationUsesStdinAsAFile {
		stdinBuffer = &bytes.Buffer{}
		if _, err := stdinBuffer.ReadFrom(env.stdin()); err != nil {
			return 0, wrapErrorwithSourceLocf(err, "prebuffering stdin")
		}
		subprocStdin = stdinBuffer
	} else {
		subprocStdin = env.stdin()
	}

	stdoutBuffer := &bytes.Buffer{}
	stderrBuffer := &bytes.Buffer{}
	exitCode, err = wrapSubprocessErrorWithSourceLoc(originalCmd,
		env.run(originalCmd, subprocStdin, stdoutBuffer, stderrBuffer))
	if err != nil {
		return 0, err
	}

	autocrashAllowed := preds.allowInConfigure || !isInConfigureStage(env)
	crash := autocrashAllowed && preds.shouldAutocrash(env, cfg, originalCmd, compilerExecInfo{
		exitCode: exitCode,
		stdout:   stdoutBuffer.Bytes(),
		stderr:   stderrBuffer.Bytes(),
	})
	if !crash {
		stdoutBuffer.WriteTo(env.stdout())
		stderrBuffer.WriteTo(env.stderr())
		return exitCode, nil
	}

	fmt.Fprintln(env.stderr(), "** Autocrash requested; crashing the compiler...**")

	// `stdinBuffer == nil` implies that `-` wasn't used as a flag.  If `-` isn't used as a
	// flag, clang will ignore stdin. We want to write our #pragma to stdin, since we can't
	// reasonably modify the files we're currently compiling.
	if stdinBuffer == nil {
		newArgs := []string{}
		// Clang can't handle `-o ${target}` when handed multiple input files. Since
		// we expect to crash before emitting anything, remove `-o ${file}` entirely.
		for i, e := 0, len(originalCmd.Args); i < e; i++ {
			a := originalCmd.Args[i]
			if a == "-o" {
				// Skip the -o here, then skip the following arg in the loop header.
				i++
			} else {
				newArgs = append(newArgs, a)
			}
		}
		// And now add args that instruct clang to read from stdin. In this case, we also
		// need to tell Clang what language the file is written in; C is as good as anything
		// for this.
		originalCmd.Args = append(newArgs, "-x", "c", "-")
		stdinBuffer = &bytes.Buffer{}
	}

	stdinBuffer.WriteString(autocrashProgramLine)
	return wrapSubprocessErrorWithSourceLoc(originalCmd,
		env.run(originalCmd, stdinBuffer, env.stdout(), env.stderr()))
}
