// Copyright 2019 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var updateGoldenFiles = flag.Bool("updategolden", false, "update golden files")
var filterGoldenTests = flag.String("rungolden", "", "regex filter for golden tests to run")

type goldenFile struct {
	Name    string         `json:"name"`
	Records []goldenRecord `json:"records"`
}

type goldenRecord struct {
	Wd  string   `json:"wd"`
	Env []string `json:"env,omitempty"`
	// runGoldenRecords will read cmd and fill
	// stdout, stderr, exitCode.
	WrapperCmd commandResult `json:"wrapper"`
	// runGoldenRecords will read stdout, stderr, err
	// and fill cmd
	Cmds []commandResult `json:"cmds"`
}

func newGoldenCmd(path string, args ...string) commandResult {
	return commandResult{
		Cmd: &command{
			Path: path,
			Args: args,
		},
	}
}

var okResult = commandResult{}
var okResults = []commandResult{okResult}
var errorResult = commandResult{
	ExitCode: 1,
	Stderr:   "someerror",
	Stdout:   "somemessage",
}
var errorResults = []commandResult{errorResult}

func runGoldenRecords(ctx *testContext, goldenDir string, files []goldenFile) {
	if filterPattern := *filterGoldenTests; filterPattern != "" {
		files = filterGoldenRecords(filterPattern, files)
	}
	if len(files) == 0 {
		ctx.t.Errorf("No goldenrecords given.")
		return
	}
	files = fillGoldenResults(ctx, files)
	if *updateGoldenFiles {
		log.Printf("updating golden files under %s", goldenDir)
		if err := os.MkdirAll(goldenDir, 0777); err != nil {
			ctx.t.Fatal(err)
		}
		for _, file := range files {
			fileHandle, err := os.Create(filepath.Join(goldenDir, file.Name))
			if err != nil {
				ctx.t.Fatal(err)
			}
			defer fileHandle.Close()

			writeGoldenRecords(ctx, fileHandle, file.Records)
		}
	} else {
		for _, file := range files {
			compareBuffer := &bytes.Buffer{}
			writeGoldenRecords(ctx, compareBuffer, file.Records)
			filePath := filepath.Join(goldenDir, file.Name)
			goldenFileData, err := os.ReadFile(filePath)
			if err != nil {
				ctx.t.Error(err)
				continue
			}
			if !bytes.Equal(compareBuffer.Bytes(), goldenFileData) {
				ctx.t.Errorf("Commands don't match the golden file under %s. Please regenerate via -updategolden to check the differences.",
					filePath)
			}
		}
	}
}

func filterGoldenRecords(pattern string, files []goldenFile) []goldenFile {
	matcher := regexp.MustCompile(pattern)
	newFiles := []goldenFile{}
	for _, file := range files {
		newRecords := []goldenRecord{}
		for _, record := range file.Records {
			cmd := record.WrapperCmd.Cmd
			str := strings.Join(append(append(record.Env, cmd.Path), cmd.Args...), " ")
			if matcher.MatchString(str) {
				newRecords = append(newRecords, record)
			}
		}
		file.Records = newRecords
		newFiles = append(newFiles, file)
	}
	return newFiles
}

func fillGoldenResults(ctx *testContext, files []goldenFile) []goldenFile {
	newFiles := []goldenFile{}
	for _, file := range files {
		newRecords := []goldenRecord{}
		for _, record := range file.Records {
			newCmds := []commandResult{}
			ctx.cmdMock = func(cmd *command, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
				if len(newCmds) >= len(record.Cmds) {
					ctx.t.Errorf("Not enough commands specified for wrapperCmd %#v and env %#v. Expected: %#v",
						record.WrapperCmd.Cmd, record.Env, record.Cmds)
					return nil
				}
				cmdResult := record.Cmds[len(newCmds)]
				cmdResult.Cmd = cmd
				if numEnvUpdates := len(cmdResult.Cmd.EnvUpdates); numEnvUpdates > 0 {
					if strings.HasPrefix(cmdResult.Cmd.EnvUpdates[numEnvUpdates-1], "PYTHONPATH") {
						cmdResult.Cmd.EnvUpdates[numEnvUpdates-1] = "PYTHONPATH=/somepath/test_binary"
					}
				}
				newCmds = append(newCmds, cmdResult)
				io.WriteString(stdout, cmdResult.Stdout)
				io.WriteString(stderr, cmdResult.Stderr)
				if cmdResult.ExitCode != 0 {
					return newExitCodeError(cmdResult.ExitCode)
				}
				return nil
			}
			ctx.stdoutBuffer.Reset()
			ctx.stderrBuffer.Reset()
			ctx.env = record.Env
			if record.Wd == "" {
				record.Wd = ctx.tempDir
			}
			ctx.wd = record.Wd
			// Create an empty wrapper at the given path.
			// Needed as we are resolving symlinks which stats the wrapper file.
			ctx.writeFile(record.WrapperCmd.Cmd.Path, "")
			record.WrapperCmd.ExitCode = callCompiler(ctx, ctx.cfg, record.WrapperCmd.Cmd)
			if hasInternalError(ctx.stderrString()) {
				ctx.t.Errorf("found an internal error for wrapperCmd %#v and env #%v. Got: %s",
					record.WrapperCmd.Cmd, record.Env, ctx.stderrString())
			}
			if len(newCmds) < len(record.Cmds) {
				ctx.t.Errorf("Too many commands specified for wrapperCmd %#v and env %#v. Expected: %#v",
					record.WrapperCmd.Cmd, record.Env, record.Cmds)
			}
			record.Cmds = newCmds
			record.WrapperCmd.Stdout = ctx.stdoutString()
			record.WrapperCmd.Stderr = ctx.stderrString()
			newRecords = append(newRecords, record)
		}
		file.Records = newRecords
		newFiles = append(newFiles, file)
	}
	return newFiles
}

func writeGoldenRecords(ctx *testContext, writer io.Writer, records []goldenRecord) {
	// We need to rewrite /tmp/${test_specific_tmpdir} records as /tmp/stable, so it's
	// deterministic across reruns. Round-trip this through JSON so there's no need to maintain
	// logic that hunts through `record`s. A side-benefit of round-tripping through a JSON `map`
	// is that `encoding/json` sorts JSON map keys, and `cros format` complains if keys aren't
	// sorted.
	encoded, err := json.Marshal(records)
	if err != nil {
		ctx.t.Fatal(err)
	}

	decoded := interface{}(nil)
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		ctx.t.Fatal(err)
	}

	stableTempDir := filepath.Join(filepath.Dir(ctx.tempDir), "stable")
	decoded, err = dfsJSONValues(decoded, func(i interface{}) interface{} {
		asString, ok := i.(string)
		if !ok {
			return i
		}
		return strings.ReplaceAll(asString, ctx.tempDir, stableTempDir)
	})
	if err != nil {
		ctx.t.Fatalf("transforming JSON values: %v", err)
	}

	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(decoded); err != nil {
		ctx.t.Fatal(err)
	}
}

// Performs a DFS on `decodedJSON`, replacing elements with the result of calling `mapFunc()` on
// each value. Only returns an error if an element type is unexpected (read: the input JSON should
// only contain the types listed for unmarshalling as an interface value here
// https://pkg.go.dev/encoding/json#Unmarshal).
//
// Two subtleties:
//  1. This calls `mapFunc()` on nested values after the transformation of their individual elements.
//     Moreover, given the JSON `[1, 2]` and a mapFunc that just returns nil, the mapFunc will be
//     called as `mapFunc(1)`, then `mapFunc(2)`, then `mapFunc({}interface{nil, nil})`.
//  2. This is not called directly on keys in maps. If you want to transform keys, you may do so when
//     `mapFunc` is called on a `map[string]interface{}`. This is to make differentiating between
//     keys and values easier.
func dfsJSONValues(decodedJSON interface{}, mapFunc func(interface{}) interface{}) (interface{}, error) {
	if decodedJSON == nil {
		return mapFunc(nil), nil
	}

	switch d := decodedJSON.(type) {
	case bool, float64, string:
		return mapFunc(decodedJSON), nil

	case []interface{}:
		newSlice := make([]interface{}, len(d))
		for i, elem := range d {
			transformed, err := dfsJSONValues(elem, mapFunc)
			if err != nil {
				return nil, err
			}
			newSlice[i] = transformed
		}
		return mapFunc(newSlice), nil

	case map[string]interface{}:
		newMap := make(map[string]interface{}, len(d))
		for k, v := range d {
			transformed, err := dfsJSONValues(v, mapFunc)
			if err != nil {
				return nil, err
			}
			newMap[k] = transformed
		}
		return mapFunc(newMap), nil

	default:
		return nil, fmt.Errorf("unexpected type in JSON: %T", decodedJSON)
	}
}
