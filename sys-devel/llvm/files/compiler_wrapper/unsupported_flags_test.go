// Copyright 2019 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"testing"
)

func TestErrorOnFstatCheckFlag(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		stderr := ctx.mustFail(callCompiler(ctx, ctx.cfg,
			ctx.newCommand(gccX86_64, "-fstack-check", mainCc)))
		if err := verifyNonInternalError(stderr,
			`option "-fstack-check" is not supported; crbug/485492`); err != nil {
			t.Fatal(err)
		}
	})
}
