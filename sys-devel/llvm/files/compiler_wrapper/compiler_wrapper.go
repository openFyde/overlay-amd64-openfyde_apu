// Copyright 2019 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	clangCrashArtifactsSubdir = "toolchain/clang_crash_diagnostics"
	crosArtifactsEnvVar       = "CROS_ARTIFACTS_TMP_DIR"

	retryableBugRetryLimit = 25
)

func callCompiler(env env, cfg *config, inputCmd *command) int {
	var compilerErr error

	if !filepath.IsAbs(inputCmd.Path) && !strings.HasPrefix(inputCmd.Path, ".") &&
		!strings.ContainsRune(inputCmd.Path, filepath.Separator) {
		if resolvedPath, err := resolveAgainstPathEnv(env, inputCmd.Path); err == nil {
			inputCmd = &command{
				Path:       resolvedPath,
				Args:       inputCmd.Args,
				EnvUpdates: inputCmd.EnvUpdates,
			}
		} else {
			compilerErr = err
		}
	}
	exitCode := 0
	if compilerErr == nil {
		exitCode, compilerErr = callCompilerInternal(env, cfg, inputCmd)
	}
	if compilerErr != nil {
		printCompilerError(env.stderr(), compilerErr)
		exitCode = 1
	}
	return exitCode
}

func containsTracesOfRetryableBug(env env, stdoutOrStderr []byte) bool {
	if containsTracesOfKernelBug(stdoutOrStderr) {
		return true
	}

	// b/417125943: temporarily work around crashes in fcp builds. Since bazel filters the
	// environment, determine whether to retry using the current directory.
	if strings.Contains(env.getwd(), "/sys-cluster/fcp") {
		return bytes.Contains(stdoutOrStderr, []byte("PLEASE submit a bug report to "))
	}
	return false
}

// Given the main builder path and the absolute path to our wrapper, returns the path to the
// 'real' compiler we should invoke.
func calculateAndroidWrapperPath(mainBuilderPath string, absWrapperPath string) string {
	// FIXME: This combination of using the directory of the symlink but the basename of the
	// link target is strange but is the logic that old android wrapper uses. Change this to use
	// directory and basename either from the absWrapperPath or from the builder.path, but don't
	// mix anymore.

	// We need to be careful here: path.Join Clean()s its result, so `./foo` will get
	// transformed to `foo`, which isn't good since we're passing this path to exec.
	basePart := filepath.Base(absWrapperPath) + "-real"
	if !strings.ContainsRune(mainBuilderPath, filepath.Separator) {
		return basePart
	}

	dirPart := filepath.Dir(mainBuilderPath)
	if cleanResult := filepath.Join(dirPart, basePart); strings.ContainsRune(cleanResult, filepath.Separator) {
		return cleanResult
	}

	return "." + string(filepath.Separator) + basePart
}

func runAndroidClangTidy(env env, cmd *command) error {
	timeout, found := env.getenv("TIDY_TIMEOUT")
	if !found {
		return env.exec(cmd)
	}
	seconds, err := strconv.Atoi(timeout)
	if err != nil || seconds == 0 {
		return env.exec(cmd)
	}
	getSourceFile := func() string {
		// Note: This depends on Android build system's clang-tidy command line format.
		// Last non-flag before "--" in cmd.Args is used as the source file name.
		sourceFile := "unknown_file"
		for _, arg := range cmd.Args {
			if arg == "--" {
				break
			}
			if strings.HasPrefix(arg, "-") {
				continue
			}
			sourceFile = arg
		}
		return sourceFile
	}
	startTime := time.Now()
	err = env.runWithTimeout(cmd, time.Duration(seconds)*time.Second)
	if !errors.Is(err, context.DeadlineExceeded) {
		// When used time is over half of TIDY_TIMEOUT, give a warning.
		// These warnings allow users to fix slow jobs before they get worse.
		usedSeconds := int(time.Since(startTime) / time.Second)
		if usedSeconds > seconds/2 {
			warning := "%s:1:1: warning: clang-tidy used %d seconds.\n"
			fmt.Fprintf(env.stdout(), warning, getSourceFile(), usedSeconds)
		}
		return err
	}
	// When DeadllineExceeded, print warning messages.
	warning := "%s:1:1: warning: clang-tidy aborted after %d seconds.\n"
	fmt.Fprintf(env.stdout(), warning, getSourceFile(), seconds)
	fmt.Fprintf(env.stdout(), "TIMEOUT: %s %s\n", cmd.Path, strings.Join(cmd.Args, " "))
	// Do not stop Android build. Just give a warning and return no error.
	return nil
}

func detectCrashArtifactsDir(env env, cfg *config) string {
	if cfg.isAndroidWrapper {
		return ""
	}

	tmpdir, ok := env.getenv(crosArtifactsEnvVar)
	if !ok {
		return ""
	}
	return filepath.Join(tmpdir, clangCrashArtifactsSubdir)
}

func callCompilerInternal(env env, cfg *config, inputCmd *command) (exitCode int, err error) {
	if err := checkUnsupportedFlags(inputCmd); err != nil {
		return 0, err
	}
	mainBuilder, err := newCommandBuilder(env, cfg, inputCmd)
	if err != nil {
		return 0, err
	}
	processPrintConfigFlag(mainBuilder)
	processPrintCmdlineFlag(mainBuilder)
	env = mainBuilder.env
	var compilerCmd *command
	disableWerrorConfig := processForceDisableWerrorFlag(env, cfg, mainBuilder)
	clangSyntax := processClangSyntaxFlag(mainBuilder)

	if err := processPerPackageFlags(cfg, mainBuilder); err != nil {
		return 0, err
	}

	rusageEnabled := isRusageEnabled(env)

	// Disable CCache for rusage logs
	// Note: Disabling Goma causes timeout related INFRA_FAILUREs in builders
	allowCCache := !rusageEnabled
	remoteBuildUsed := false

	workAroundRetryableBug := false
	if cfg.isAndroidWrapper {
		mainBuilder.path = calculateAndroidWrapperPath(mainBuilder.path, mainBuilder.absWrapperPath)
		switch mainBuilder.target.compilerType {
		case clangType:
			mainBuilder.addPreUserArgs(mainBuilder.cfg.clangFlags...)
			mainBuilder.addPreUserArgs(mainBuilder.cfg.commonFlags...)
			mainBuilder.addPostUserArgs(mainBuilder.cfg.clangPostFlags...)
			inheritGomaFromEnv := true
			// Android doesn't support rewrapper; don't try to use it.
			if remoteBuildUsed, err = processGomaCccFlags(mainBuilder, inheritGomaFromEnv); err != nil {
				return 0, err
			}
			compilerCmd = mainBuilder.build()
		case clangTidyType:
			compilerCmd = mainBuilder.build()
		default:
			return 0, newErrorwithSourceLocf("unsupported compiler: %s", mainBuilder.target.compiler)
		}
	} else {
		if isRiscvBuildWithoutAckFlag(env, mainBuilder) {
			return 0, errors.New(riscvExperimentalUseError)
		}

		cSrcFile, tidyFlags, tidyMode := processClangTidyFlags(mainBuilder)
		crashArtifactsDir := detectCrashArtifactsDir(env, cfg)
		if mainBuilder.target.compilerType == clangType {
			err := prepareClangCommand(crashArtifactsDir, mainBuilder)
			if err != nil {
				return 0, err
			}
			if tidyMode != tidyModeNone {
				allowCCache = false
				// Remove and ignore goma flags.
				_, err := removeOneUserCmdlineFlagWithValue(mainBuilder, "--gomacc-path")
				if err != nil && err != errNoSuchCmdlineArg {
					return 0, err
				}

				clangCmdWithoutRemoteBuildAndCCache := mainBuilder.build()

				switch tidyMode {
				case tidyModeTricium:
					err = runClangTidyForTricium(env, clangCmdWithoutRemoteBuildAndCCache, cSrcFile, tidyFlags, crashArtifactsDir)
				case tidyModeAll:
					err = runClangTidy(env, clangCmdWithoutRemoteBuildAndCCache, cSrcFile, tidyFlags)
				default:
					panic(fmt.Sprintf("Unknown tidy mode: %v", tidyMode))
				}

				if err != nil {
					return 0, err
				}
			}

			if remoteBuildUsed, err = processRemoteBuildAndCCacheFlags(allowCCache, mainBuilder); err != nil {
				return 0, err
			}
			compilerCmd = mainBuilder.build()
		} else {
			if clangSyntax {
				allowCCache = false
				_, clangCmd, err := calcClangCommand(crashArtifactsDir, allowCCache, mainBuilder.clone())
				if err != nil {
					return 0, err
				}
				_, gccCmd, err := calcGccCommand(rusageEnabled, mainBuilder)
				if err != nil {
					return 0, err
				}
				return checkClangSyntax(env, clangCmd, gccCmd)
			}
			remoteBuildUsed, compilerCmd, err = calcGccCommand(rusageEnabled, mainBuilder)
			if err != nil {
				return 0, err
			}
			workAroundRetryableBug = true
		}
	}

	// If builds matching some heuristic should crash, crash them. Since this is purely a
	// debugging tool, don't offer any nice features with it (e.g., rusage, ...).
	if shouldUseCrashBuildsHeuristic && mainBuilder.target.compilerType == clangType {
		return buildWithAutocrash(env, cfg, compilerCmd)
	}

	if rusageEnabled {
		compilerCmd = removeRusageFromCommand(compilerCmd)
	}

	if disableWerrorConfig.enabled {
		return doubleBuildWithWNoError(env, cfg, compilerCmd, disableWerrorConfig)
	}
	if shouldCompileWithFallback(env) {
		if rusageEnabled {
			return 0, newUserErrorf("TOOLCHAIN_RUSAGE_OUTPUT is meaningless with ANDROID_LLVM_PREBUILT_COMPILER_PATH")
		}
		return compileWithFallback(env, cfg, compilerCmd, mainBuilder.absWrapperPath)
	}

	errRetryCompilation := errors.New("compilation retry requested")
	var runCompiler func(willLogRusage bool) (int, error)
	if !workAroundRetryableBug {
		runCompiler = func(willLogRusage bool) (int, error) {
			var err error
			if willLogRusage {
				err = env.run(compilerCmd, env.stdin(), env.stdout(), env.stderr())
			} else if cfg.isAndroidWrapper && mainBuilder.target.compilerType == clangTidyType {
				// Only clang-tidy has timeout feature now.
				err = runAndroidClangTidy(env, compilerCmd)
			} else {
				// Note: We return from this in non-fatal circumstances only if the
				// underlying env is not really doing an exec, e.g. commandRecordingEnv.
				err = env.exec(compilerCmd)
			}
			return wrapSubprocessErrorWithSourceLoc(compilerCmd, err)
		}
	} else {
		getStdin, err := prebufferStdinIfNeeded(env, compilerCmd)
		if err != nil {
			return 0, wrapErrorwithSourceLocf(err, "prebuffering stdin: %v", err)
		}

		stdoutBuffer := &bytes.Buffer{}
		stderrBuffer := &bytes.Buffer{}
		retryAttempt := 0
		runCompiler = func(willLogRusage bool) (int, error) {
			retryAttempt++
			stdoutBuffer.Reset()
			stderrBuffer.Reset()

			exitCode, compilerErr := wrapSubprocessErrorWithSourceLoc(compilerCmd,
				env.run(compilerCmd, getStdin(), stdoutBuffer, stderrBuffer))

			if compilerErr != nil || exitCode != 0 {
				if retryAttempt < retryableBugRetryLimit && (errorContainsTracesOfKernelBug(compilerErr) ||
					containsTracesOfRetryableBug(env, stdoutBuffer.Bytes()) || containsTracesOfRetryableBug(env, stderrBuffer.Bytes())) {
					return exitCode, errRetryCompilation
				}
			}
			_, stdoutErr := stdoutBuffer.WriteTo(env.stdout())
			_, stderrErr := stderrBuffer.WriteTo(env.stderr())
			if stdoutErr != nil {
				return exitCode, wrapErrorwithSourceLocf(err, "writing stdout: %v", stdoutErr)
			}
			if stderrErr != nil {
				return exitCode, wrapErrorwithSourceLocf(err, "writing stderr: %v", stderrErr)
			}
			return exitCode, compilerErr
		}
	}

	for {
		var exitCode int
		commitRusage, err := maybeCaptureRusage(env, compilerCmd, func(willLogRusage bool) error {
			var err error
			exitCode, err = runCompiler(willLogRusage)
			return err
		})

		switch {
		case err == errRetryCompilation:
			// Loop around again.
		case err != nil:
			return exitCode, err
		default:
			if !remoteBuildUsed {
				if err := commitRusage(exitCode); err != nil {
					return exitCode, fmt.Errorf("commiting rusage: %v", err)
				}
			}
			return exitCode, err
		}
	}
}

func hasUserArg(argName string, builder *commandBuilder) bool {
	for _, argValue := range builder.args {
		if strings.Contains(argValue.value, argName) && argValue.fromUser {
			return true
		}
	}
	return false
}

func prepareClangCommand(crashArtifactsDir string, builder *commandBuilder) (err error) {
	if !builder.cfg.isHostWrapper {
		processSysrootFlag(builder)
	}
	builder.addPreUserArgs(builder.cfg.clangFlags...)

	var crashDiagFlagName = "-fcrash-diagnostics-dir"
	if crashArtifactsDir != "" &&
		!hasUserArg(crashDiagFlagName, builder) {
		builder.addPreUserArgs(crashDiagFlagName + "=" + crashArtifactsDir)
	}

	builder.addPostUserArgs(builder.cfg.clangPostFlags...)
	calcCommonPreUserArgs(builder)
	return processClangFlags(builder)
}

func calcClangCommand(crashArtifactsDir string, allowCCache bool, builder *commandBuilder) (bool, *command, error) {
	err := prepareClangCommand(crashArtifactsDir, builder)
	if err != nil {
		return false, nil, err
	}
	remoteBuildUsed, err := processRemoteBuildAndCCacheFlags(allowCCache, builder)
	if err != nil {
		return remoteBuildUsed, nil, err
	}
	return remoteBuildUsed, builder.build(), nil
}

func calcGccCommand(enableRusage bool, builder *commandBuilder) (bool, *command, error) {
	if !builder.cfg.isHostWrapper {
		processSysrootFlag(builder)
	}
	builder.addPreUserArgs(builder.cfg.gccFlags...)
	calcCommonPreUserArgs(builder)
	processGccFlags(builder)

	remoteBuildUsed, err := processRemoteBuildAndCCacheFlags(!enableRusage, builder)
	if err != nil {
		return remoteBuildUsed, nil, err
	}
	return remoteBuildUsed, builder.build(), nil
}

func calcCommonPreUserArgs(builder *commandBuilder) {
	builder.addPreUserArgs(builder.cfg.commonFlags...)
	if !builder.cfg.isHostWrapper {
		processLibGCCFlags(builder)
		processThumbCodeFlags(builder)
		processStackProtectorFlags(builder)
		processX86Flags(builder)
	}
	processSanitizerFlags(builder)
}

func processRemoteBuildAndCCacheFlags(allowCCache bool, builder *commandBuilder) (remoteBuildUsed bool, err error) {
	remoteBuildUsed, err = processRemoteBuildFlags(builder)
	if err != nil {
		return remoteBuildUsed, err
	}
	if !remoteBuildUsed && allowCCache {
		processCCacheFlag(builder)
	}
	return remoteBuildUsed, nil
}

func getAbsWrapperPath(env env, wrapperCmd *command) (string, error) {
	wrapperPath := getAbsCmdPath(env, wrapperCmd)
	evaledCmdPath, err := filepath.EvalSymlinks(wrapperPath)
	if err != nil {
		return "", wrapErrorwithSourceLocf(err, "failed to evaluate symlinks for %s", wrapperPath)
	}
	return evaledCmdPath, nil
}

func printCompilerError(writer io.Writer, compilerErr error) {
	if _, ok := compilerErr.(userError); ok {
		fmt.Fprintf(writer, "%s\n", compilerErr)
	} else {
		emailAccount := "chromeos-toolchain"
		if isAndroidConfig() {
			emailAccount = "android-llvm"
		}
		fmt.Fprintf(writer,
			"Internal error. Please report to %s@google.com.\n%s\n",
			emailAccount, compilerErr)
	}
}

func needStdinTee(inputCmd *command) bool {
	lastArg := ""
	for _, arg := range inputCmd.Args {
		if arg == "-" && lastArg != "-o" {
			return true
		}
		lastArg = arg
	}
	return false
}

func prebufferStdinIfNeeded(env env, inputCmd *command) (getStdin func() io.Reader, err error) {
	// We pre-buffer the entirety of stdin, since the compiler may exit mid-invocation with an
	// error, which may leave stdin partially read.
	if !needStdinTee(inputCmd) {
		// This won't produce deterministic input to the compiler, but stdin shouldn't
		// matter in this case, so...
		return env.stdin, nil
	}

	stdinBuffer := &bytes.Buffer{}
	if _, err := stdinBuffer.ReadFrom(env.stdin()); err != nil {
		return nil, wrapErrorwithSourceLocf(err, "prebuffering stdin")
	}

	return func() io.Reader { return bytes.NewReader(stdinBuffer.Bytes()) }, nil
}
