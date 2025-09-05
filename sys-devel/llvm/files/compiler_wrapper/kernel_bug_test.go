// Copyright 2021 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
package main

import (
	"errors"
	"io"
	"testing"
)

func getErrorIndicatingKernelBug() error {
	return errors.New("waitid: errno 512")
}

func TestWrapperRetriesCompilationsOnApparentKernelBugsSurfacedInGo(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		ctx.cmdMock = func(cmd *command, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
			switch {
			case ctx.cmdCount < retryableBugRetryLimit:
				return getErrorIndicatingKernelBug()

			case ctx.cmdCount == retryableBugRetryLimit:
				return nil

			default:
				t.Fatalf("unexpected command: %#v", cmd)
				return nil
			}
		}
		ctx.must(callCompiler(ctx, ctx.cfg, ctx.newCommand(gccX86_64, mainCc)))
		if ctx.cmdCount != retryableBugRetryLimit {
			t.Errorf("expected %d retries. Got: %d", retryableBugRetryLimit, ctx.cmdCount)
		}
	})
}

func TestWrapperRetriesCompilationsOnApparentKernelBugsSurfacedInGCC(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		ctx.cmdMock = func(cmd *command, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
			if ctx.cmdCount >= retryableBugRetryLimit {
				return nil
			}
			_, err := io.WriteString(stderr, "fatal error: failed to get exit status: Unknown error 512")
			if err != nil {
				t.Fatalf("Failed writing to stdout: %v", err)
			}
			return newExitCodeError(1)
		}
		ctx.must(callCompiler(ctx, ctx.cfg, ctx.newCommand(gccX86_64, mainCc)))
		if ctx.cmdCount != retryableBugRetryLimit {
			t.Errorf("expected %d retries. Got: %d", retryableBugRetryLimit, ctx.cmdCount)
		}
	})
}

func TestWrapperOnlyRetriesCompilationAFiniteNumberOfTimes(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		kernelBugErr := getErrorIndicatingKernelBug()
		ctx.cmdMock = func(cmd *command, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
			if ctx.cmdCount > retryableBugRetryLimit {
				t.Fatal("command count exceeded kernel bug retry limit; infinite loop?")
			}
			return kernelBugErr
		}
		stderr := ctx.mustFail(callCompiler(ctx, ctx.cfg, ctx.newCommand(gccX86_64, mainCc)))
		if err := verifyInternalError(stderr); err != nil {
			t.Errorf("Internal error wasn't reported: %v", err)
		}
		if ctx.cmdCount != retryableBugRetryLimit {
			t.Errorf("expected %d retries. Got: %d", retryableBugRetryLimit, ctx.cmdCount)
		}
	})
}
