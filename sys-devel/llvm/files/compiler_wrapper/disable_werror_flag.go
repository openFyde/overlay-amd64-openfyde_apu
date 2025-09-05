// Copyright 2019 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
)

const numWErrorEstimate = 30

func getForceDisableWerrorDir(env env, cfg *config) string {
	return path.Join(getCompilerArtifactsDir(env), "toolchain/fatal_clang_warnings")
}

type forceDisableWerrorConfig struct {
	// If reportToStdout is true, we'll write -Werror reports to stdout. Otherwise, they'll be
	// written to reportDir. If reportDir is empty, it will be determined via
	// `getForceDisableWerrorDir`.
	//
	// Neither of these have specified values if `enabled == false`.
	reportDir      string
	reportToStdout bool

	// If true, `-Werror` reporting should be used.
	enabled bool
}

func processForceDisableWerrorFlag(env env, cfg *config, builder *commandBuilder) forceDisableWerrorConfig {
	if cfg.isAndroidWrapper {
		return forceDisableWerrorConfig{
			reportToStdout: true,
			enabled:        cfg.useLlvmNext,
		}
	}

	// CrOS supports two modes for enabling this flag:
	// 1 (preferred). A CFLAG that specifies the directory to write reports to. e.g.,
	//   `-D_CROSTC_FORCE_DISABLE_WERROR=/path/to/directory`. This flag will be removed from the
	//   command before the compiler is invoked. If multiple of these are passed, the last one
	//   wins, but all are removed from the build command.
	// 2 (dispreferred, but supported). An environment variable, FORCE_DISABLE_WERROR, set to
	//   any nonempty value. In this case, the wrapper will write to either
	//   ${CROS_ARTIFACTS_TMP_DIR}/toolchain/fatal_clang_warnings, or to
	//   /tmp/toolchain/fatal_clang_warnings.
	//
	// Two modes are supported because some ebuilds filter the env, while others will filter
	// CFLAGS. Vanishingly few (none?) filter both, though.
	const cflagPrefix = "-D_CROSTC_FORCE_DISABLE_WERROR="

	argDir := ""
	sawArg := false
	builder.transformArgs(func(arg builderArg) string {
		value := arg.value
		if !strings.HasPrefix(value, cflagPrefix) {
			return value
		}
		argDir = value[len(cflagPrefix):]
		sawArg = true
		return ""
	})

	// CrOS only wants this functionality to apply to clang, though flags should also be removed
	// for GCC.
	if builder.target.compilerType != clangType {
		return forceDisableWerrorConfig{enabled: false}
	}

	if sawArg {
		return forceDisableWerrorConfig{
			reportDir: argDir,
			// Skip this when in src_configure: some build systems ignore CFLAGS
			// modifications after configure, so this flag must be specified before
			// src_configure, but we only want the flag to apply to actual builds.
			enabled: !isInConfigureStage(env),
		}
	}

	envValue, _ := env.getenv("FORCE_DISABLE_WERROR")
	return forceDisableWerrorConfig{enabled: envValue != ""}
}

func disableWerrorFlags(originalArgs, extraFlags []string) []string {
	allExtraFlags := append([]string{}, extraFlags...)
	newArgs := make([]string, 0, len(originalArgs)+numWErrorEstimate)
	for _, flag := range originalArgs {
		if strings.HasPrefix(flag, "-Werror=") {
			allExtraFlags = append(allExtraFlags, strings.Replace(flag, "-Werror", "-Wno-error", 1))
		}
		if !strings.Contains(flag, "-warnings-as-errors") {
			newArgs = append(newArgs, flag)
		}
	}
	return append(newArgs, allExtraFlags...)
}

func isLikelyAConfTest(env env, cfg *config, cmd *command) bool {
	// Android doesn't do mid-build `configure`s, so we don't need to worry about this there.
	if cfg.isAndroidWrapper {
		return false
	}

	// Ignore anything that's likely to be a cmake configuration step. These put the compiler
	// into a TryCompile dir.
	if strings.Contains(env.getwd(), "CMakeFiles/CMakeScratch/TryCompile-") {
		return true
	}

	wasLastArgOutput := false
	for _, a := range cmd.Args {
		// The kernel, for example, will do configure tests with /dev/null as a source file.
		if a == "/dev/null" || strings.HasPrefix(a, "conftest.c") {
			return true
		}

		// b/417950454: scons conftests run during src_compile at times, but consistently use
		// `-o ${some_dir}/.sconf.temp/conftest_${hash}_${num}.o`.
		if wasLastArgOutput {
			if strings.HasSuffix(a, ".o") && strings.Contains(a, "/.sconf.temp/conftest_") {
				return true
			}
			wasLastArgOutput = false
		} else {
			wasLastArgOutput = a == "-o"
		}
	}
	return false
}

func getWnoErrorFlags(stdout, stderr []byte) []string {
	needWnoError := false
	extraFlags := []string{}
	for _, submatches := range regexp.MustCompile(`error:.* \[(-W[^\]]+)\]`).FindAllSubmatch(stderr, -1) {
		bracketedMatch := submatches[1]

		// Some warnings are promoted to errors by -Werror. These contain `-Werror` in the
		// brackets specifying the warning name. A broad, follow-up `-Wno-error` should
		// disable those.
		//
		// _Others_ are implicitly already errors, and will not be disabled by `-Wno-error`.
		// These do not have `-Wno-error` in their brackets. These need to explicitly have
		// `-Wno-error=${warning_name}`. See b/325463152 for an example.
		if bytes.HasPrefix(bracketedMatch, []byte("-Werror,")) || bytes.HasSuffix(bracketedMatch, []byte(",-Werror")) {
			needWnoError = true
		} else {
			// In this case, the entire bracketed match is the warning flag. Trim the
			// first two chars off to account for the `-W` matched in the regex.
			warningName := string(bracketedMatch[2:])
			extraFlags = append(extraFlags, "-Wno-error="+warningName)
		}
	}
	needWnoError = needWnoError || bytes.Contains(stdout, []byte("warnings-as-errors")) || bytes.Contains(stdout, []byte("clang-diagnostic-"))

	if len(extraFlags) == 0 && !needWnoError {
		return nil
	}
	return append(extraFlags, "-Wno-error")
}

func doubleBuildWithWNoError(env env, cfg *config, originalCmd *command, werrorConfig forceDisableWerrorConfig) (exitCode int, err error) {
	originalStdoutBuffer := &bytes.Buffer{}
	originalStderrBuffer := &bytes.Buffer{}
	// TODO: This is a bug in the old wrapper that it drops the ccache path
	// during double build. Fix this once we don't compare to the old wrapper anymore.
	if originalCmd.Path == "/usr/bin/ccache" {
		originalCmd.Path = "ccache"
	}

	getStdin, err := prebufferStdinIfNeeded(env, originalCmd)
	if err != nil {
		return 0, wrapErrorwithSourceLocf(err, "prebuffering stdin: %v", err)
	}

	var originalExitCode int
	commitOriginalRusage, err := maybeCaptureRusage(env, originalCmd, func(willLogRusage bool) error {
		originalExitCode, err = wrapSubprocessErrorWithSourceLoc(originalCmd,
			env.run(originalCmd, getStdin(), originalStdoutBuffer, originalStderrBuffer))
		return err
	})
	if err != nil {
		return 0, err
	}

	// The only way we can do anything useful is if it looks like the failure
	// was -Werror-related.
	retryWithExtraFlags := []string{}
	if originalExitCode != 0 && !isLikelyAConfTest(env, cfg, originalCmd) {
		retryWithExtraFlags = getWnoErrorFlags(originalStdoutBuffer.Bytes(), originalStderrBuffer.Bytes())
	}
	if len(retryWithExtraFlags) == 0 {
		if err := commitOriginalRusage(originalExitCode); err != nil {
			return 0, fmt.Errorf("commiting rusage: %v", err)
		}
		originalStdoutBuffer.WriteTo(env.stdout())
		originalStderrBuffer.WriteTo(env.stderr())
		return originalExitCode, nil
	}

	retryStdoutBuffer := &bytes.Buffer{}
	retryStderrBuffer := &bytes.Buffer{}
	retryCommand := &command{
		Path:       originalCmd.Path,
		Args:       disableWerrorFlags(originalCmd.Args, retryWithExtraFlags),
		EnvUpdates: originalCmd.EnvUpdates,
	}

	var retryExitCode int
	commitRetryRusage, err := maybeCaptureRusage(env, retryCommand, func(willLogRusage bool) error {
		retryExitCode, err = wrapSubprocessErrorWithSourceLoc(retryCommand,
			env.run(retryCommand, getStdin(), retryStdoutBuffer, retryStderrBuffer))
		return err
	})
	if err != nil {
		return 0, err
	}

	// If -Wno-error fixed us, pretend that we never ran without -Wno-error. Otherwise, pretend
	// that we never ran the second invocation.
	if retryExitCode != 0 {
		originalStdoutBuffer.WriteTo(env.stdout())
		originalStderrBuffer.WriteTo(env.stderr())
		if err := commitOriginalRusage(originalExitCode); err != nil {
			return 0, fmt.Errorf("commiting rusage: %v", err)
		}
		return originalExitCode, nil
	}

	if err := commitRetryRusage(retryExitCode); err != nil {
		return 0, fmt.Errorf("commiting rusage: %v", err)
	}

	retryStdoutBuffer.WriteTo(env.stdout())
	retryStderrBuffer.WriteTo(env.stderr())

	lines := []string{}
	if originalStderrBuffer.Len() > 0 {
		lines = append(lines, originalStderrBuffer.String())
	}
	if originalStdoutBuffer.Len() > 0 {
		lines = append(lines, originalStdoutBuffer.String())
	}
	outputToLog := strings.Join(lines, "\n")

	// Ignore the error here; we can't do anything about it. The result is always valid (though
	// perhaps incomplete) even if this returns an error.
	parentProcesses, _ := collectAllParentProcesses()
	jsonData := warningsJSONData{
		Cwd:             env.getwd(),
		Command:         append([]string{originalCmd.Path}, originalCmd.Args...),
		Stdout:          outputToLog,
		ParentProcesses: parentProcesses,
	}

	// Write warning report to stdout for Android.  On Android,
	// double-build can be requested on remote builds as well, where there
	// is no canonical place to write the warnings report.
	if werrorConfig.reportToStdout {
		stdout := env.stdout()
		io.WriteString(stdout, "<LLVM_NEXT_ERROR_REPORT>")
		if err := json.NewEncoder(stdout).Encode(jsonData); err != nil {
			return 0, wrapErrorwithSourceLocf(err, "error in json.Marshal")
		}
		io.WriteString(stdout, "</LLVM_NEXT_ERROR_REPORT>")
		return retryExitCode, nil
	}

	// All of the below is basically logging. If we fail at any point, it's
	// reasonable for that to fail the build. This is all meant for FYI-like
	// builders in the first place.

	// Buildbots use a nonzero umask, which isn't quite what we want: these directories should
	// be world-readable and world-writable.
	oldMask := env.umask(0)
	defer env.umask(oldMask)

	reportDir := werrorConfig.reportDir
	if reportDir == "" {
		reportDir = getForceDisableWerrorDir(env, cfg)
	}

	// Allow root and regular users to write to this without issue.
	if err := os.MkdirAll(reportDir, 0777); err != nil {
		return 0, wrapErrorwithSourceLocf(err, "error creating warnings directory %s", reportDir)
	}

	// Have some tag to show that files aren't fully written. It would be sad if
	// an interrupted build (or out of disk space, or similar) caused tools to
	// have to be overly-defensive.
	const incompleteSuffix = ".incomplete"

	// Coming up with a consistent name for this is difficult (compiler command's
	// SHA can clash in the case of identically named files in different
	// directories, or similar); let's use a random one.
	tmpFile, err := os.CreateTemp(reportDir, "warnings_report*.json"+incompleteSuffix)
	if err != nil {
		return 0, wrapErrorwithSourceLocf(err, "error creating warnings file")
	}

	if err := tmpFile.Chmod(0666); err != nil {
		return 0, wrapErrorwithSourceLocf(err, "error chmoding the file to be world-readable/writeable")
	}

	enc := json.NewEncoder(tmpFile)
	if err := enc.Encode(jsonData); err != nil {
		_ = tmpFile.Close()
		return 0, wrapErrorwithSourceLocf(err, "error writing warnings data")
	}

	if err := tmpFile.Close(); err != nil {
		return 0, wrapErrorwithSourceLocf(err, "error closing warnings file")
	}

	if err := os.Rename(tmpFile.Name(), tmpFile.Name()[:len(tmpFile.Name())-len(incompleteSuffix)]); err != nil {
		return 0, wrapErrorwithSourceLocf(err, "error removing incomplete suffix from warnings file")
	}

	return retryExitCode, nil
}

func parseParentPidFromPidStat(pidStatContents string) (parentPid int, ok bool) {
	// The parent's pid is the fourth field of /proc/[pid]/stat. Sadly, the second field can
	// have spaces in it. It ends at the last ')' in the contents of /proc/[pid]/stat.
	lastParen := strings.LastIndex(pidStatContents, ")")
	if lastParen == -1 {
		return 0, false
	}

	thirdFieldAndBeyond := strings.TrimSpace(pidStatContents[lastParen+1:])
	fields := strings.Fields(thirdFieldAndBeyond)
	if len(fields) < 2 {
		return 0, false
	}

	fourthField := fields[1]
	parentPid, err := strconv.Atoi(fourthField)
	if err != nil {
		return 0, false
	}
	return parentPid, true
}

func collectProcessData(pid int) (args, env []string, parentPid int, err error) {
	procDir := fmt.Sprintf("/proc/%d", pid)

	readFile := func(fileName string) (string, error) {
		s, err := os.ReadFile(path.Join(procDir, fileName))
		if err != nil {
			return "", fmt.Errorf("reading %s: %v", fileName, err)
		}
		return string(s), nil
	}

	statStr, err := readFile("stat")
	if err != nil {
		return nil, nil, 0, err
	}

	parentPid, ok := parseParentPidFromPidStat(statStr)
	if !ok {
		return nil, nil, 0, fmt.Errorf("no parseable parent PID found in %q", statStr)
	}

	argsStr, err := readFile("cmdline")
	if err != nil {
		return nil, nil, 0, err
	}
	args = strings.Split(argsStr, "\x00")

	envStr, err := readFile("environ")
	if err != nil {
		return nil, nil, 0, err
	}
	env = strings.Split(envStr, "\x00")
	return args, env, parentPid, nil
}

// The returned []processData is valid even if this returns an error. The error is just the first we
// encountered when trying to collect parent process data.
func collectAllParentProcesses() ([]processData, error) {
	results := []processData{}
	for parent := os.Getppid(); parent != 1; {
		args, env, p, err := collectProcessData(parent)
		if err != nil {
			return results, fmt.Errorf("inspecting parent %d: %v", parent, err)
		}
		results = append(results, processData{Args: args, Env: env})
		parent = p
	}
	return results, nil
}

type processData struct {
	Args []string `json:"invocation"`
	Env  []string `json:"env"`
}

// Struct used to write JSON. Fields have to be uppercase for the json encoder to read them.
type warningsJSONData struct {
	Cwd             string        `json:"cwd"`
	Command         []string      `json:"command"`
	Stdout          string        `json:"stdout"`
	ParentProcesses []processData `json:"parent_process_data"`
}
