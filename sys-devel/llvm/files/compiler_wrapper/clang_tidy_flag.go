// Copyright 2019 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type useTidyMode int

const clangTidyCrashSubstring = "PLEASE submit a bug report"

const (
	tidyModeNone useTidyMode = iota
	tidyModeAll
	tidyModeTricium
)

func processClangTidyFlags(builder *commandBuilder) (cSrcFile string, clangTidyFlags []string, mode useTidyMode) {
	builder.transformArgs(func(arg builderArg) string {
		const prefix = "-clang-tidy-flag="
		if !strings.HasPrefix(arg.value, prefix) {
			return arg.value
		}

		clangTidyFlags = append(clangTidyFlags, arg.value[len(prefix):])
		return ""
	})

	withTidy, _ := builder.env.getenv("WITH_TIDY")
	if withTidy == "" {
		return "", clangTidyFlags, tidyModeNone
	}
	srcFileSuffixes := []string{
		".c",
		".cc",
		".cpp",
		".C",
		".cxx",
		".c++",
	}
	cSrcFile = ""
	srcSuffix := ""
	lastArg := ""
	for _, arg := range builder.args {
		if lastArg != "-o" {
			for _, suffix := range srcFileSuffixes {
				if strings.HasSuffix(arg.value, suffix) {
					srcSuffix = suffix
					cSrcFile = arg.value
					break
				}
			}
		}
		lastArg = arg.value
	}

	if cSrcFile == "" {
		return "", clangTidyFlags, tidyModeNone
	}

	if withTidy == "tricium" {
		// Files generated from protobufs can result in _many_ clang-tidy complaints, and aren't
		// worth linting in general. Don't.
		if strings.HasSuffix(cSrcFile, ".pb"+srcSuffix) {
			mode = tidyModeNone
		} else {
			mode = tidyModeTricium
		}
	} else {
		mode = tidyModeAll
	}
	return cSrcFile, clangTidyFlags, mode
}

func calcClangTidyInvocation(env env, clangCmd *command, cSrcFile string, tidyFlags ...string) (*command, error) {
	resourceDir, err := getClangResourceDir(env, clangCmd.Path)
	if err != nil {
		return nil, err
	}

	clangTidyPath := filepath.Join(filepath.Dir(clangCmd.Path), "clang-tidy")
	args := append([]string{}, tidyFlags...)
	args = append(args, cSrcFile, "--", "-resource-dir="+resourceDir)
	args = append(args, clangCmd.Args...)
	return &command{
		Path:       clangTidyPath,
		Args:       args,
		EnvUpdates: clangCmd.EnvUpdates,
	}, nil
}

func runClangTidyForTricium(env env, clangCmd *command, cSrcFile string, extraTidyFlags []string, crashArtifactsDir string) error {
	fixesDir := filepath.Join(getCompilerArtifactsDir(env), "linting-output", "clang-tidy")
	if err := os.MkdirAll(fixesDir, 0777); err != nil {
		return fmt.Errorf("creating fixes directory at %q: %v", fixesDir, err)
	}

	f, err := os.CreateTemp(fixesDir, "lints-")
	if err != nil {
		return fmt.Errorf("making tempfile for tidy: %v", err)
	}
	f.Close()

	// `f` is an 'anchor'; it ensures we won't create a similarly-named file in the future.
	// Hence, we can't delete it.
	fixesFilePath := f.Name() + ".yaml"
	fixesMetadataPath := f.Name() + ".json"

	extraTidyFlags = append(extraTidyFlags, "--export-fixes="+fixesFilePath, "--header-filter=.*")
	clangTidyCmd, err := calcClangTidyInvocation(env, clangCmd, cSrcFile, extraTidyFlags...)
	if err != nil {
		return fmt.Errorf("calculating tidy invocation: %v", err)
	}

	stdstreams := &strings.Builder{}
	// Note: We pass nil as stdin as we checked before that the compiler
	// was invoked with a source file argument.
	exitCode, err := wrapSubprocessErrorWithSourceLoc(clangTidyCmd,
		env.run(clangTidyCmd, nil, stdstreams, stdstreams))
	if err != nil {
		return err
	}

	type crashOutput struct {
		CrashReproducerPath string `json:"crash_reproducer_path"`
		Stdstreams          string `json:"stdstreams"`
	}

	type metadata struct {
		Args        []string     `json:"args"`
		CrashOutput *crashOutput `json:"crash_output"`
		Executable  string       `json:"executable"`
		ExitCode    int          `json:"exit_code"`
		LintTarget  string       `json:"lint_target"`
		Stdstreams  string       `json:"stdstreams"`
		Wd          string       `json:"wd"`
	}

	meta := &metadata{
		Args:        clangTidyCmd.Args,
		CrashOutput: nil,
		Executable:  clangTidyCmd.Path,
		ExitCode:    exitCode,
		LintTarget:  cSrcFile,
		Stdstreams:  stdstreams.String(),
		Wd:          env.getwd(),
	}

	// Sometimes, clang-tidy crashes. Unfortunately, these don't get funnelled through the
	// standard clang crash machinery. :(. Try to work with our own.
	if crashArtifactsDir != "" && strings.Contains(meta.Stdstreams, clangTidyCrashSubstring) {
		tidyCrashArtifacts := path.Join(crashArtifactsDir, "clang-tidy")
		if err := os.MkdirAll(tidyCrashArtifacts, 0777); err != nil {
			return fmt.Errorf("creating crash artifacts directory at %q: %v", tidyCrashArtifacts, err)
		}

		f, err := os.CreateTemp(tidyCrashArtifacts, "crash-")
		if err != nil {
			return fmt.Errorf("making tempfile for crash output: %v", err)
		}
		f.Close()

		reproCmd := &command{}
		*reproCmd = *clangCmd
		reproCmd.Args = append(reproCmd.Args, "-E", "-o", f.Name())

		reproOut := &strings.Builder{}
		_, err = wrapSubprocessErrorWithSourceLoc(reproCmd, env.run(reproCmd, nil, reproOut, reproOut))
		if err != nil {
			return fmt.Errorf("attempting to produce a clang-tidy crash reproducer: %v", err)
		}
		meta.CrashOutput = &crashOutput{
			CrashReproducerPath: f.Name(),
			Stdstreams:          reproOut.String(),
		}
	}

	f, err = os.Create(fixesMetadataPath)
	if err != nil {
		return fmt.Errorf("creating fixes metadata: %v", err)
	}

	if err := json.NewEncoder(f).Encode(meta); err != nil {
		return fmt.Errorf("writing fixes metadata: %v", err)
	}

	if err := f.Close(); err != nil {
		return fmt.Errorf("finalizing fixes metadata: %v", err)
	}
	return nil
}

func runClangTidy(env env, clangCmd *command, cSrcFile string, extraTidyFlags []string) error {
	extraTidyFlags = append(extraTidyFlags,
		"-checks="+strings.Join([]string{
			"*",
			"-bugprone-narrowing-conversions",
			"-cppcoreguidelines-*",
			"-fuchsia-*",
			"-google-readability*",
			"-google-runtime-references",
			"-hicpp-*",
			"-llvm-*",
			"-misc-non-private-member-variables-in-classes",
			"-misc-unused-parameters",
			"-modernize-*",
			"-readability-*",
		}, ","))
	clangTidyCmd, err := calcClangTidyInvocation(env, clangCmd, cSrcFile, extraTidyFlags...)
	if err != nil {
		return fmt.Errorf("calculating clang-tidy invocation: %v", err)
	}

	// Note: We pass nil as stdin as we checked before that the compiler
	// was invoked with a source file argument.
	exitCode, err := wrapSubprocessErrorWithSourceLoc(clangTidyCmd,
		env.run(clangTidyCmd, nil, env.stdout(), env.stderr()))
	if err == nil && exitCode != 0 {
		// Note: We continue on purpose when clang-tidy fails
		// to maintain compatibility with the previous wrapper.
		fmt.Fprint(env.stderr(), "clang-tidy failed")
	}
	return err
}
