// Copyright 2022 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestBuildWithAutoCrashDoesNothingIfCrashIsNotRequested(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		neverAutoCrash := buildWithAutocrashPredicates{
			allowInConfigure: true,
			shouldAutocrash: func(env, *config, *command, compilerExecInfo) bool {
				return false
			},
		}

		exitCode, err := buildWithAutocrashImpl(ctx, ctx.cfg, ctx.newCommand(clangX86_64, mainCc), neverAutoCrash)
		if err != nil {
			t.Fatalf("unexpectedly failed with %v", err)
		}
		ctx.must(exitCode)
		if ctx.cmdCount != 1 {
			t.Errorf("expected 1 call. Got: %d", ctx.cmdCount)
		}
	})
}

func TestBuildWithAutoCrashSkipsAutocrashLogicIfInConfigureAndConfigureChecksDisabled(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		alwaysAutocrash := buildWithAutocrashPredicates{
			allowInConfigure: false,
			shouldAutocrash: func(env, *config, *command, compilerExecInfo) bool {
				return true
			},
		}

		ctx.env = append(ctx.env, "EBUILD_PHASE=configure")
		exitCode, err := buildWithAutocrashImpl(ctx, ctx.cfg, ctx.newCommand(clangX86_64, mainCc), alwaysAutocrash)
		if err != nil {
			t.Fatalf("unexpectedly failed with %v", err)
		}
		ctx.must(exitCode)
		if ctx.cmdCount != 1 {
			t.Errorf("expected 1 call. Got: %d", ctx.cmdCount)
		}
	})
}

func TestBuildWithAutoCrashRerunsIfPredicateRequestsCrash(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		autocrashPostCmd := buildWithAutocrashPredicates{
			allowInConfigure: true,
			shouldAutocrash: func(env, *config, *command, compilerExecInfo) bool {
				return true
			},
		}

		ctx.cmdMock = func(cmd *command, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
			hasDash := false
			for _, arg := range cmd.Args {
				if arg == "-" {
					hasDash = true
					break
				}
			}

			switch ctx.cmdCount {
			case 1:
				if hasDash {
					t.Error("Got `-` on command 1; didn't want that.")
				}
				return nil
			case 2:
				if !hasDash {
					t.Error("Didn't get `-` on command 2; wanted that.")
				} else {
					input := stdin.(*bytes.Buffer)
					if s := input.String(); !strings.Contains(s, autocrashProgramLine) {
						t.Errorf("Input was %q; expected %q to be in it", s, autocrashProgramLine)
					}
				}
				return nil
			default:
				t.Fatalf("Unexpected command count: %d", ctx.cmdCount)
				panic("Unreachable")
			}
		}

		exitCode, err := buildWithAutocrashImpl(ctx, ctx.cfg, ctx.newCommand(clangX86_64, mainCc), autocrashPostCmd)
		if err != nil {
			t.Fatalf("unexpectedly failed with %v", err)
		}
		ctx.must(exitCode)

		if ctx.cmdCount != 2 {
			t.Errorf("expected 2 calls. Got: %d", ctx.cmdCount)
		}
	})
}

func TestBuildWithAutoCrashAddsDashAndWritesToStdinIfInputFileIsNotStdin(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		autocrashPostCmd := buildWithAutocrashPredicates{
			allowInConfigure: true,
			shouldAutocrash: func(env, *config, *command, compilerExecInfo) bool {
				return true
			},
		}

		ctx.cmdMock = func(cmd *command, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
			numDashes := 0
			for _, arg := range cmd.Args {
				if arg == "-" {
					numDashes++
				}
			}

			switch ctx.cmdCount {
			case 1:
				if numDashes != 0 {
					t.Errorf("Got %d dashes on command 1; want 0", numDashes)
				}
				return nil
			case 2:
				if numDashes != 1 {
					t.Errorf("Got %d dashes on command 2; want 1", numDashes)
				}

				input := stdin.(*bytes.Buffer).String()
				stdinHasAutocrashLine := strings.Contains(input, autocrashProgramLine)
				if !stdinHasAutocrashLine {
					t.Error("Got no autocrash line on the second command; wanted that")
				}
				return nil
			default:
				t.Fatalf("Unexpected command count: %d", ctx.cmdCount)
				panic("Unreachable")
			}
		}

		exitCode, err := buildWithAutocrashImpl(ctx, ctx.cfg, ctx.newCommand(clangX86_64, mainCc), autocrashPostCmd)
		if err != nil {
			t.Fatalf("unexpectedly failed with %v", err)
		}
		ctx.must(exitCode)

		if ctx.cmdCount != 2 {
			t.Errorf("expected 2 calls. Got: %d", ctx.cmdCount)
		}
	})
}

func TestBuildWithAutoCrashAppendsToStdinIfStdinIsTheOnlyInputFile(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		autocrashPostCmd := buildWithAutocrashPredicates{
			allowInConfigure: true,
			shouldAutocrash: func(env, *config, *command, compilerExecInfo) bool {
				return true
			},
		}

		ctx.cmdMock = func(cmd *command, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
			numDashes := 0
			for _, arg := range cmd.Args {
				if arg == "-" {
					numDashes++
				}
			}

			if numDashes != 1 {
				t.Errorf("Got %d dashes on command %d (args: %#v); want 1", numDashes, ctx.cmdCount, cmd.Args)
			}

			input := stdin.(*bytes.Buffer).String()
			stdinHasAutocrashLine := strings.Contains(input, autocrashProgramLine)

			switch ctx.cmdCount {
			case 1:
				if stdinHasAutocrashLine {
					t.Error("Got autocrash line on the first command; did not want that")
				}
				return nil
			case 2:
				if !stdinHasAutocrashLine {
					t.Error("Got no autocrash line on the second command; wanted that")
				}
				return nil
			default:
				t.Fatalf("Unexpected command count: %d", ctx.cmdCount)
				panic("Unreachable")
			}
		}

		exitCode, err := buildWithAutocrashImpl(ctx, ctx.cfg, ctx.newCommand(clangX86_64, "-x", "c", "-"), autocrashPostCmd)
		if err != nil {
			t.Fatalf("unexpectedly failed with %v", err)
		}
		ctx.must(exitCode)

		if ctx.cmdCount != 2 {
			t.Errorf("expected 2 calls. Got: %d", ctx.cmdCount)
		}
	})
}

func TestCrashBuildFiltersObjectFileOptionOnCrashes(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		autocrashPostCmd := buildWithAutocrashPredicates{
			allowInConfigure: true,
			shouldAutocrash: func(env, *config, *command, compilerExecInfo) bool {
				return true
			},
		}

		const outputFileName = "/path/to/foo.o"

		ctx.cmdMock = func(cmd *command, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
			cmdOutputArg := (*string)(nil)
			for i, e := range cmd.Args {
				if e == "-o" {
					// Assume something follows. If not, we'll crash and the
					// test will fail.
					cmdOutputArg = &cmd.Args[i+1]
				}
			}

			switch ctx.cmdCount {
			case 1:
				if cmdOutputArg == nil || *cmdOutputArg != outputFileName {
					t.Errorf("Got command args %q; want `-o %q` in them", cmd.Args, outputFileName)
				}
				return nil
			case 2:
				if cmdOutputArg != nil {
					t.Errorf("Got command args %q; want no mention of `-o %q` in them", cmd.Args, outputFileName)
				}
				return nil
			default:
				t.Fatalf("Unexpected command count: %d", ctx.cmdCount)
				panic("Unreachable")
			}
		}

		exitCode, err := buildWithAutocrashImpl(ctx, ctx.cfg, ctx.newCommand(clangX86_64, "-o", outputFileName, mainCc), autocrashPostCmd)
		if err != nil {
			t.Fatalf("unexpectedly failed with %v", err)
		}
		ctx.must(exitCode)

		if ctx.cmdCount != 2 {
			t.Errorf("expected 2 calls. Got: %d", ctx.cmdCount)
		}
	})
}
