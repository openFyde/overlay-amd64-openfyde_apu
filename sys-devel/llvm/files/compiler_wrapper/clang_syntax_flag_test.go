// Copyright 2019 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"errors"
	"fmt"
	"io"
	"path"
	"strings"
	"testing"
)

func TestCheckClangSyntaxByNestedCall(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		ctx.cmdMock = func(cmd *command, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
			if ctx.cmdCount == 1 {
				if err := verifyPath(cmd, "usr/bin/clang"); err != nil {
					return err
				}
				if err := verifyArgOrder(cmd, mainCc, "-fsyntax-only", `-stdlib=libstdc\+\+`); err != nil {
					return err
				}
			}
			return nil
		}
		cmd := ctx.must(callCompiler(ctx, ctx.cfg,
			ctx.newCommand(gccX86_64, "-clang-syntax", mainCc)))
		if ctx.cmdCount != 2 {
			t.Errorf("expected 2 calls. Got: %d", ctx.cmdCount)
		}
		if err := verifyPath(cmd, gccX86_64+".real"); err != nil {
			t.Error(err)
		}
		if err := verifyArgCount(cmd, 0, "-clang-syntax"); err != nil {
			t.Error(err)
		}
	})
}

func TestForwardStdOutAndStderrFromClangSyntaxCheck(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		ctx.cmdMock = func(cmd *command, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
			if ctx.cmdCount == 1 {
				fmt.Fprint(stdout, "somemessage")
				fmt.Fprint(stderr, "someerror")
			}
			return nil
		}
		ctx.must(callCompiler(ctx, ctx.cfg,
			ctx.newCommand(gccX86_64, "-clang-syntax", mainCc)))
		if ctx.stdoutString() != "somemessage" {
			t.Errorf("stdout was not forwarded. Got: %s", ctx.stdoutString())
		}
		if ctx.stderrString() != "someerror" {
			t.Errorf("stderr was not forwarded. Got: %s", ctx.stderrString())
		}
	})
}

func TestForwardStdinToClangSyntaxCheck(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		ctx.cmdMock = func(cmd *command, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
			// Note: This is called for the clang syntax call as well as for
			// the gcc call, and we assert that stdin is cloned and forwarded
			// to both.
			stdinStr := ctx.readAllString(stdin)
			if stdinStr != "someinput" {
				return fmt.Errorf("unexpected stdin. Got: %s", stdinStr)
			}
			return nil
		}
		io.WriteString(&ctx.stdinBuffer, "someinput")
		ctx.must(callCompiler(ctx, ctx.cfg,
			ctx.newCommand(gccX86_64, "-clang-syntax", "-", mainCc)))
	})
}

func TestForwardExitCodeFromClangSyntaxCheck(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		ctx.cmdMock = func(cmd *command, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
			if ctx.cmdCount == 1 {
				return newExitCodeError(23)
			}
			return nil
		}
		exitCode := callCompiler(ctx, ctx.cfg,
			ctx.newCommand(gccX86_64, "-clang-syntax", mainCc))
		if exitCode != 23 {
			t.Errorf("unexpected exit code. Got: %d", exitCode)
		}
	})
}

func TestReportGeneralErrorsFromClangSyntaxCheck(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		ctx.cmdMock = func(cmd *command, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
			if ctx.cmdCount == 1 {
				return errors.New("someerror")
			}
			return nil
		}
		stderr := ctx.mustFail(callCompiler(ctx, ctx.cfg,
			ctx.newCommand(gccX86_64, "-clang-syntax", mainCc)))
		if err := verifyInternalError(stderr); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(stderr, "someerror") {
			t.Errorf("unexpected error. Got: %s", stderr)
		}
	})
}

func TestIgnoreClangSyntaxCheckWhenCallingClang(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		ctx.cmdMock = func(cmd *command, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
			if ctx.cmdCount > 1 {
				return fmt.Errorf("Unexpected call %#v", cmd)
			}
			return nil
		}
		cmd := ctx.must(callCompiler(ctx, ctx.cfg,
			ctx.newCommand(clangX86_64, "-clang-syntax", mainCc)))
		if err := verifyArgCount(cmd, 0, "-clang-syntax"); err != nil {
			t.Error(err)
		}
	})
}

func TestUseGomaForClangSyntaxCheck(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		gomaPath := path.Join(ctx.tempDir, "gomacc")
		// Create a file so the gomacc path is valid.
		ctx.writeFile(gomaPath, "")
		ctx.env = []string{"GOMACC_PATH=" + gomaPath}
		ctx.cmdMock = func(cmd *command, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
			if ctx.cmdCount == 1 {
				if err := verifyPath(cmd, gomaPath); err != nil {
					return err
				}
				if err := verifyArgOrder(cmd, "usr/bin/clang", mainCc); err != nil {
					return err
				}
			}
			return nil
		}
		cmd := ctx.must(callCompiler(ctx, ctx.cfg,
			ctx.newCommand(gccX86_64, "-clang-syntax", mainCc)))
		if ctx.cmdCount != 2 {
			t.Errorf("expected 2 calls. Got: %d", ctx.cmdCount)
		}
		if err := verifyPath(cmd, gomaPath); err != nil {
			t.Error(err)
		}
	})
}

func TestPartiallyOmitCCacheForClangSyntaxCheck(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		ctx.cfg.useCCache = true
		ctx.cmdMock = func(cmd *command, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
			if ctx.cmdCount == 1 {
				if err := verifyPath(cmd, "usr/bin/clang"); err != nil {
					return err
				}
			}
			return nil
		}
		cmd := ctx.must(callCompiler(ctx, ctx.cfg,
			ctx.newCommand(gccX86_64, "-clang-syntax", mainCc)))
		if ctx.cmdCount != 2 {
			t.Errorf("expected 2 calls. Got: %d", ctx.cmdCount)
		}
		if err := verifyPath(cmd, "/usr/bin/ccache"); err != nil {
			t.Error(err)
		}
	})
}
