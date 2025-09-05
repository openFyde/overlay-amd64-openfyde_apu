// Copyright 2019 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"errors"
	"fmt"
	"syscall"
	"testing"
)

func TestNewErrorwithSourceLocfMessage(t *testing.T) {
	err := newErrorwithSourceLocf("a%sc", "b")
	if err.Error() != "errors_test.go:15: abc" {
		t.Errorf("Error message incorrect. Got: %s", err.Error())
	}
}

func TestWrapErrorwithSourceLocfMessage(t *testing.T) {
	cause := errors.New("someCause")
	err := wrapErrorwithSourceLocf(cause, "a%sc", "b")
	if err.Error() != "errors_test.go:23: abc: someCause" {
		t.Errorf("Error message incorrect. Got: %s", err.Error())
	}
}

func TestNewUserErrorf(t *testing.T) {
	err := newUserErrorf("a%sc", "b")
	if err.Error() != "abc" {
		t.Errorf("Error message incorrect. Got: %s", err.Error())
	}
}

func TestSubprocessOk(t *testing.T) {
	exitCode, err := wrapSubprocessErrorWithSourceLoc(nil, nil)
	if exitCode != 0 {
		t.Errorf("unexpected exit code. Got: %d", exitCode)
	}
	if err != nil {
		t.Errorf("unexpected error. Got: %s", err)
	}
}

func TestSubprocessExitCodeError(t *testing.T) {
	exitCode, err := wrapSubprocessErrorWithSourceLoc(nil, newExitCodeError(23))
	if exitCode != 23 {
		t.Errorf("unexpected exit code. Got: %d", exitCode)
	}
	if err != nil {
		t.Errorf("unexpected error. Got: %s", err)
	}
}

func TestSubprocessCCacheError(t *testing.T) {
	_, err := wrapSubprocessErrorWithSourceLoc(&command{Path: "/usr/bin/ccache"}, syscall.ENOENT)
	if _, ok := err.(userError); !ok {
		t.Errorf("unexpected error type. Got: %T", err)
	}
	if err.Error() != "ccache not found under /usr/bin/ccache. Please install it" {
		t.Errorf("unexpected error message. Got: %s", err)
	}
}

func TestSubprocessGeneralError(t *testing.T) {
	cmd := &command{Path: "somepath"}
	_, err := wrapSubprocessErrorWithSourceLoc(cmd, errors.New("someerror"))
	if err.Error() != fmt.Sprintf("errors_test.go:68: failed to execute %#v: someerror", cmd) {
		t.Errorf("Error message incorrect. Got: %s", err.Error())
	}
}
