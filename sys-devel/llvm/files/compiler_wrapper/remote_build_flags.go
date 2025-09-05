// Copyright 2019 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

var errNoSuchCmdlineArg = errors.New("no such commandline argument")

// Removes one flag from `builder`, assuming that a value follows the flag. Two formats are
// supported for this: `--foo=bar` and `--foo bar`. In either case, "bar" will be returned as the
// `value`.
//
// If no flag is found on the commandline, this returns the `errNoSuchCmdlineArg` error. `builder`
// is unmodified if this error is returned, but its contents are unspecified if any other error is
// returned.
//
// In the case of multiple such flags, only the first encountered will be removed.
func removeOneUserCmdlineFlagWithValue(builder *commandBuilder, flagName string) (flagValue string, err error) {
	const (
		searchingForFlag uint8 = iota
		searchingForValue
		searchComplete
	)

	flagRequiresAValue := func() error { return newUserErrorf("flag %q requires a value", flagName) }
	searchState := searchingForFlag
	builder.transformArgs(func(arg builderArg) string {
		if err != nil {
			return arg.value
		}

		switch searchState {
		case searchingForFlag:
			if !arg.fromUser {
				return arg.value
			}

			if arg.value == flagName {
				searchState = searchingForValue
				return ""
			}

			isArgEq := strings.HasPrefix(arg.value, flagName) && arg.value[len(flagName)] == '='
			if !isArgEq {
				return arg.value
			}

			flagValue = arg.value[len(flagName)+1:]
			searchState = searchComplete
			return ""

		case searchingForValue:
			if !arg.fromUser {
				err = flagRequiresAValue()
				return arg.value
			}

			flagValue = arg.value
			searchState = searchComplete
			return ""

		case searchComplete:
			return arg.value

		default:
			panic(fmt.Sprintf("unknown search state: %v", searchState))
		}
	})

	if err != nil {
		return "", err
	}

	switch searchState {
	case searchingForFlag:
		return "", errNoSuchCmdlineArg

	case searchingForValue:
		return "", flagRequiresAValue()

	case searchComplete:
		return flagValue, nil

	default:
		panic(fmt.Sprintf("unknown search state: %v", searchState))
	}
}

func processGomaCccFlags(builder *commandBuilder, inheritFromEnv bool) (gomaUsed bool, err error) {
	gomaPath, err := removeOneUserCmdlineFlagWithValue(builder, "--gomacc-path")
	if err != nil && err != errNoSuchCmdlineArg {
		return false, err
	}

	if inheritFromEnv && (err == errNoSuchCmdlineArg || gomaPath == "") {
		gomaPath, _ = builder.env.getenv("GOMACC_PATH")
	}

	if gomaPath != "" {
		if _, err := os.Lstat(gomaPath); err == nil {
			builder.wrapPath(gomaPath)
			return true, nil
		}
	}
	return false, nil
}

func processRewrapperCcFlags(builder *commandBuilder) (rewrapperUsed bool, err error) {
	rewrapperPath, pathErr := removeOneUserCmdlineFlagWithValue(builder, "--rewrapper-path")
	if pathErr != nil && pathErr != errNoSuchCmdlineArg {
		return false, err
	}

	rewrapperCfg, cfgErr := removeOneUserCmdlineFlagWithValue(builder, "--rewrapper-cfg")
	if cfgErr != nil && cfgErr != errNoSuchCmdlineArg {
		return false, err
	}

	if pathErr == errNoSuchCmdlineArg {
		if cfgErr != errNoSuchCmdlineArg {
			return false, newUserErrorf("--rewrapper-path must be specified if --rewrapper-cfg is")
		}
		return false, nil
	}

	if cfgErr == errNoSuchCmdlineArg {
		return false, newUserErrorf("--rewrapper-cfg must be specified if --rewrapper-path is")
	}

	// It's unclear that we should have a similar fallback to gomacc if --rewrapper-path doesn't
	// exist, so don't until it's obviously necessary.
	builder.wrapPath(rewrapperPath, "-cfg", rewrapperCfg)
	return true, nil
}

func processRemoteBuildFlags(builder *commandBuilder) (remoteBuildUsed bool, err error) {
	rewrapperUsed, err := processRewrapperCcFlags(builder)
	if err != nil {
		return rewrapperUsed, err
	}

	inheritGomaFromEnv := !rewrapperUsed
	gomaUsed, err := processGomaCccFlags(builder, inheritGomaFromEnv)
	remoteBuildUsed = gomaUsed || rewrapperUsed
	if err != nil {
		return remoteBuildUsed, err
	}

	if gomaUsed && rewrapperUsed {
		return true, newUserErrorf("rewrapper and gomacc are mutually exclusive")
	}
	return remoteBuildUsed, nil
}
