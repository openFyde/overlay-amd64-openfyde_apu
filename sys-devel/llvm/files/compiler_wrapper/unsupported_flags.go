// Copyright 2019 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

func checkUnsupportedFlags(cmd *command) error {
	for _, arg := range cmd.Args {
		if arg == "-fstack-check" {
			return newUserErrorf(`option %q is not supported; crbug/485492`, arg)
		}
	}
	return nil
}
