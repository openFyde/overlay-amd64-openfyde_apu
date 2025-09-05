// Copyright 2019 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"os"
	"path"
	"reflect"
	"testing"
)

func TestCommandlineFlagParsing(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		type testCase struct {
			extraFlags []string
			// If this is nonempty, expectedValue is ignored. Otherwise, expectedValue
			// has the expected value for the flag, and expectedCommand has the expected
			// (extra) flags in the builder after filtering.
			expectedError      string
			expectedValue      string
			expectedExtraFlags []string
		}

		const flagName = "--flag"
		testCases := []testCase{
			{
				extraFlags:    nil,
				expectedError: errNoSuchCmdlineArg.Error(),
			},
			{
				extraFlags:    []string{flagName + "a"},
				expectedError: errNoSuchCmdlineArg.Error(),
			},
			{
				extraFlags:    []string{flagName},
				expectedError: "flag \"" + flagName + "\" requires a value",
			},
			{
				extraFlags:         []string{flagName, "foo"},
				expectedValue:      "foo",
				expectedExtraFlags: nil,
			},
			{
				extraFlags:         []string{flagName + "=foo"},
				expectedValue:      "foo",
				expectedExtraFlags: nil,
			},
			{
				extraFlags:         []string{flagName + "="},
				expectedValue:      "",
				expectedExtraFlags: nil,
			},
			{
				extraFlags:         []string{flagName + "=foo", flagName + "=bar"},
				expectedValue:      "foo",
				expectedExtraFlags: []string{flagName + "=bar"},
			},
		}

		for _, testCase := range testCases {
			cmd := ctx.newCommand(gccX86_64, testCase.extraFlags...)
			builder, err := newCommandBuilder(ctx, ctx.cfg, cmd)
			if err != nil {
				t.Fatalf("Failed creating a command builder: %v", err)
			}

			flagValue, err := removeOneUserCmdlineFlagWithValue(builder, flagName)
			if err != nil {
				if testCase.expectedError == "" {
					t.Errorf("given extra flags %q, got unexpected error removing %q: %v", testCase.extraFlags, flagName, err)
					continue
				}

				if e := err.Error(); e != testCase.expectedError {
					t.Errorf("given extra flags %q, got error %q; wanted %q", testCase.extraFlags, e, testCase.expectedError)
				}
				continue
			}

			if testCase.expectedError != "" {
				t.Errorf("given extra flags %q, got no error, but expected %q", testCase.extraFlags, testCase.expectedError)
				continue
			}

			if flagValue != testCase.expectedValue {
				t.Errorf("given extra flags %q, got value %q, but expected %q", testCase.extraFlags, flagValue, testCase.expectedValue)
			}

			currentFlags := []string{}
			// Chop off the first arg, which should just be the compiler
			for _, a := range builder.args {
				currentFlags = append(currentFlags, a.value)
			}

			sameFlags := (len(currentFlags) == 0 && len(testCase.expectedExtraFlags) == 0) || reflect.DeepEqual(currentFlags, testCase.expectedExtraFlags)
			if !sameFlags {
				t.Errorf("given extra flags %q, got post-removal flags %q, but expected %q", testCase.extraFlags, currentFlags, testCase.expectedExtraFlags)
			}
		}
	})
}

func TestCallGomaccIfEnvIsGivenAndValid(t *testing.T) {
	withGomaccTestContext(t, func(ctx *testContext, gomaPath string) {
		ctx.env = []string{"GOMACC_PATH=" + gomaPath}
		cmd := ctx.must(callCompiler(ctx, ctx.cfg,
			ctx.newCommand(gccX86_64, mainCc)))
		if err := verifyPath(cmd, gomaPath); err != nil {
			t.Error(err)
		}
		if err := verifyArgOrder(cmd, gccX86_64+".real", mainCc); err != nil {
			t.Error(err)
		}
	})
}

func TestOmitGomaccIfEnvIsGivenButInvalid(t *testing.T) {
	withGomaccTestContext(t, func(ctx *testContext, gomaPath string) {
		if err := os.Remove(gomaPath); err != nil {
			t.Fatalf("failed removing fake goma file at %q: %v", gomaPath, err)
		}

		ctx.env = []string{"GOMACC_PATH=" + gomaPath}
		cmd := ctx.must(callCompiler(ctx, ctx.cfg,
			ctx.newCommand(gccX86_64, mainCc)))
		if err := verifyPath(cmd, gccX86_64+".real"); err != nil {
			t.Error(err)
		}
	})
}

func TestCallGomaccIfArgIsGivenAndValid(t *testing.T) {
	withGomaccTestContext(t, func(ctx *testContext, gomaPath string) {
		cmd := ctx.must(callCompiler(ctx, ctx.cfg,
			ctx.newCommand(gccX86_64, mainCc, "--gomacc-path", gomaPath)))
		if err := verifyPath(cmd, gomaPath); err != nil {
			t.Error(err)
		}
		if err := verifyArgCount(cmd, 0, "--gomacc-path"); err != nil {
			t.Error(err)
		}
		if err := verifyArgCount(cmd, 0, gomaPath); err != nil {
			t.Error(err)
		}
		if err := verifyArgOrder(cmd, gccX86_64+".real", mainCc); err != nil {
			t.Error(err)
		}
	})
}

func TestOmitGomaccIfArgIsGivenButInvalid(t *testing.T) {
	withGomaccTestContext(t, func(ctx *testContext, gomaPath string) {
		if err := os.Remove(gomaPath); err != nil {
			t.Fatalf("failed removing fake goma file at %q: %v", gomaPath, err)
		}

		cmd := ctx.must(callCompiler(ctx, ctx.cfg,
			ctx.newCommand(gccX86_64, mainCc, "--gomacc-path", gomaPath)))
		if err := verifyPath(cmd, gccX86_64+".real"); err != nil {
			t.Error(err)
		}
	})
}

func TestErrorOnGomaccArgWithoutValue(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		stderr := ctx.mustFail(callCompiler(ctx, ctx.cfg,
			ctx.newCommand(gccX86_64, mainCc, "--gomacc-path")))
		if err := verifyNonInternalError(stderr, "flag \"--gomacc-path\" requires a value"); err != nil {
			t.Error(err)
		}
	})
}

func TestOmitGomaccByDefault(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		cmd := ctx.must(callCompiler(ctx, ctx.cfg,
			ctx.newCommand(gccX86_64, mainCc)))
		if err := verifyPath(cmd, gccX86_64+".real"); err != nil {
			t.Error(err)
		}
	})
}

func withGomaccTestContext(t *testing.T, f func(*testContext, string)) {
	withTestContext(t, func(ctx *testContext) {
		gomaPath := path.Join(ctx.tempDir, "gomacc")
		// Create a file so the gomacc path is valid.
		ctx.writeFile(gomaPath, "")
		f(ctx, gomaPath)
	})
}

func TestRewrapperDefersToTheWrapperProperly(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		cmd := ctx.must(callCompiler(ctx, ctx.cfg,
			ctx.newCommand(gccX86_64, mainCc, "--rewrapper-path", "/rewrapper", "--rewrapper-cfg", "/some-cfg", "some", "other", "args")))
		if err := verifyPath(cmd, "/rewrapper"); err != nil {
			t.Error(err)
		}
		if err := verifyArgOrder(cmd, "-cfg", "/some-cfg", gccX86_64+".real", mainCc, "some", "other", "args"); err != nil {
			t.Error(err)
		}
	})
}

func TestRewrapperCfgMustBePrsentIfRewrapperPathIs(t *testing.T) {
	withGomaccTestContext(t, func(ctx *testContext, gomaPath string) {
		stderr := ctx.mustFail(callCompiler(ctx, ctx.cfg,
			ctx.newCommand(gccX86_64, mainCc, "--rewrapper-path", "/rewrapper")))
		if err := verifyNonInternalError(stderr, "--rewrapper-cfg must be specified if --rewrapper-path is"); err != nil {
			t.Error(err)
		}
	})
}

func TestRewrapperPathMustBePrsentIfRewrapperCfgIs(t *testing.T) {
	withGomaccTestContext(t, func(ctx *testContext, gomaPath string) {
		stderr := ctx.mustFail(callCompiler(ctx, ctx.cfg,
			ctx.newCommand(gccX86_64, mainCc, "--rewrapper-cfg", "/some-cfg")))
		if err := verifyNonInternalError(stderr, "--rewrapper-path must be specified if --rewrapper-cfg is"); err != nil {
			t.Error(err)
		}
	})
}

func TestRewrapperAndGomaAreMutuallyExclusive(t *testing.T) {
	withGomaccTestContext(t, func(ctx *testContext, gomaPath string) {
		stderr := ctx.mustFail(callCompiler(ctx, ctx.cfg,
			ctx.newCommand(gccX86_64, mainCc, "--rewrapper-path", "/rewrapper", "--rewrapper-cfg", "/some-cfg", "--gomacc-path", gomaPath)))
		if err := verifyNonInternalError(stderr, "rewrapper and gomacc are mutually exclusive"); err != nil {
			t.Error(err)
		}
	})
}

func TestRewrapperBlocksGomaInheritanceFromEnv(t *testing.T) {
	withGomaccTestContext(t, func(ctx *testContext, gomaPath string) {
		ctx.env = []string{"GOMACC_PATH=" + gomaPath}
		cmd := ctx.must(callCompiler(ctx, ctx.cfg,
			ctx.newCommand(gccX86_64, mainCc, "--rewrapper-path", "/rewrapper", "--rewrapper-cfg", "/some-cfg")))
		if err := verifyPath(cmd, "/rewrapper"); err != nil {
			t.Error(err)
		}
		if err := verifyArgOrder(cmd, "-cfg", "/some-cfg", gccX86_64+".real", mainCc); err != nil {
			t.Error(err)
		}
	})
}
