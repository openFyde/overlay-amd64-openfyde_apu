// Copyright 2019 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"testing"
)

const crosNonHardenedGoldenDir = "testdata/cros_nonhardened_golden"

func TestCrosNonHardenedConfig(t *testing.T) {
	withTestContext(t, func(ctx *testContext) {
		ctx.NoteTestWritesToUmask()

		useLlvmNext := false
		useCCache := true
		cfg, err := getConfig("cros.nonhardened", useCCache, useLlvmNext, "123")
		if err != nil {
			t.Fatal(err)
		}
		ctx.updateConfig(cfg)

		runGoldenRecords(ctx, crosNonHardenedGoldenDir, createSyswrapperGoldenInputs(ctx))
	})
}
