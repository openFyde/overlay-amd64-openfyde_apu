// Copyright 2019 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"bytes"
	"os"
	"path"
	"path/filepath"
	"strings"
)

func processClangFlags(builder *commandBuilder) error {
	env := builder.env
	clangDir := ""
	if builder.cfg.isHostWrapper {
		clangDir = filepath.Dir(builder.absWrapperPath)
	} else {
		clangDir = filepath.Join(builder.rootPath, "usr/bin/")
		if !filepath.IsAbs(builder.path) {
			// If sysroot_wrapper is invoked by relative path, call actual compiler in
			// relative form. This is necessary to remove absolute path from compile
			// outputs.
			var err error
			clangDir, err = filepath.Rel(env.getwd(), clangDir)
			if err != nil {
				return wrapErrorwithSourceLocf(err, "failed to make clangDir %s relative to %s.", clangDir, env.getwd())
			}
		}
	}

	clangBasename := "clang"
	if strings.HasSuffix(builder.target.compiler, "++") {
		clangBasename = "clang++"
	}

	// Unsupported flags to remove from the clang command line.
	// Use of -Qunused-arguments allows this set to be small, just those
	// that clang still warns about.
	unsupported := map[string]bool{
		"-Xcompiler":     true,
		"-avoid-version": true,
	}

	unsupportedPrefixes := []string{"-Wstrict-aliasing=", "-finline-limit="}

	// clang with '-ftrapv' generates 'call __mulodi4', which is only implemented
	// in compiler-rt library. However compiler-rt library only has i386/x86_64
	// backends (see '/usr/lib/clang/3.7.0/lib/linux/libclang_rt.*'). GCC, on the
	// other hand, generate 'call __mulvdi3', which is implemented in libgcc. See
	// bug chromium:503229.
	armUnsupported := map[string]bool{"-ftrapv": true}
	if builder.cfg.isHostWrapper {
		unsupported["-ftrapv"] = true
	}

	// Clang may use different options for the same or similar functionality.
	gccToClang := map[string]string{
		"-Wno-error=cpp":                 "-Wno-#warnings",
		"-Wno-error=maybe-uninitialized": "-Wno-error=uninitialized",
	}

	// Note: not using builder.transformArgs as we need to add multiple arguments
	// based on a single input argument, and also be able to return errors.
	newArgs := []builderArg{}

	// Note: Used for detecting whether we are building non-thumb2 compatible
	// baremetal arm targets. This is used to override the target triple for
	// armv6m targets, as Clang target triple normalization does not work
	// correctly when it's set to "arm" instead of "armv6m". See b/373947346.
	isArmV6M := false
	for _, arg := range builder.args {
		// Adds an argument with the given value, preserving the
		// fromUser value of the original argument.
		addNewArg := func(value string) {
			newArgs = append(newArgs, builderArg{
				fromUser: arg.fromUser,
				value:    value,
			})
		}

		if arg.value == "-march=armv6m" || strings.HasPrefix(arg.value, "-target=armv6m") || strings.HasPrefix(arg.value, "--target=armv6m") {
			isArmV6M = true
		}

		if mapped, ok := gccToClang[arg.value]; ok {
			addNewArg(mapped)
			continue
		}

		if unsupported[arg.value] {
			continue
		}

		if hasAtLeastOnePrefix(arg.value, unsupportedPrefixes) {
			continue
		}

		if builder.target.arch == "armv7a" && builder.target.sys == "linux" {
			if armUnsupported[arg.value] {
				continue
			}
		}

		if clangOnly := "-Xclang-only="; strings.HasPrefix(arg.value, clangOnly) {
			addNewArg(arg.value[len(clangOnly):])
			continue
		}

		if clangPath := "-Xclang-path="; strings.HasPrefix(arg.value, clangPath) {
			clangPathValue := arg.value[len(clangPath):]
			resourceDir, err := getClangResourceDir(env, filepath.Join(clangDir, clangBasename))
			if err != nil {
				return err
			}
			clangDir = clangPathValue

			addNewArg("-resource-dir=" + resourceDir)
			addNewArg("--gcc-toolchain=/usr")
			continue
		}

		addNewArg(arg.value)
	}
	builder.args = newArgs

	builder.path = filepath.Join(clangDir, clangBasename)

	// Specify the target for clang.
	if !builder.cfg.isHostWrapper {
		linkerPath := getLinkerPath(env, builder.target.target+"-ld.bfd", builder.rootPath)
		relLinkerPath, err := filepath.Rel(env.getwd(), linkerPath)
		if err != nil {
			return wrapErrorwithSourceLocf(err, "failed to make linker path %s relative to %s",
				linkerPath, env.getwd())
		}
		prefixPath := path.Join(relLinkerPath, builder.target.target+"-")
		builder.addPreUserArgs("--prefix=" + prefixPath)
		builder.addPostUserArgs("-B" + relLinkerPath)

		// Override for armv6m, as it doesn't support thumb2, and Clang target
		// triple normalization does not work correctly when it's set to "arm" instead
		// of "armv6m". See b/373947346.
		if isArmV6M {
			builder.addPostUserArgs("-target", "armv6m-none-eabi")
		} else {
			builder.addPostUserArgs("-target", builder.target.target)
		}
		// An lld bug related to ICF prevents us from using LLVMs addrsig sections for
		// armv7a. See b/413702636.
		if builder.target.arch == "armv7a" {
			builder.addPreUserArgs("-fno-addrsig")
		}
	}
	return nil
}

func getClangResourceDir(env env, clangPath string) (string, error) {
	readResourceCmd := &command{
		Path: clangPath,
		Args: []string{"--print-resource-dir"},
	}
	stdoutBuffer := bytes.Buffer{}
	if err := env.run(readResourceCmd, nil, &stdoutBuffer, env.stderr()); err != nil {
		return "", wrapErrorwithSourceLocf(err,
			"failed to call clang to read the resouce-dir: %#v",
			readResourceCmd)
	}
	resourceDir := strings.TrimRight(stdoutBuffer.String(), "\n")
	return resourceDir, nil
}

// Return the a directory which contains an 'ld' that gcc is using.
func getLinkerPath(env env, linkerCmd string, rootPath string) string {
	// We did not pass the tuple i686-pc-linux-gnu to x86-32 clang. Instead,
	// we passed '-m32' to clang. As a result, clang does not want to use the
	// i686-pc-linux-gnu-ld, so we need to add this to help clang find the right
	// linker.
	if linkerPath, err := resolveAgainstPathEnv(env, linkerCmd); err == nil {
		// FIXME: We are not using filepath.EvalSymlinks to only unpack
		// one layer of symlinks to match the old wrapper. Investigate
		// why this is important or simplify to filepath.EvalSymlinks.
		if fi, err := os.Lstat(linkerPath); err == nil {
			if fi.Mode()&os.ModeSymlink != 0 {
				if linkPath, err := os.Readlink(linkerPath); err == nil {
					linkerPath = linkPath
				}
			}
			return filepath.Dir(linkerPath)
		}
	}

	// When using the sdk outside chroot, we need to provide the cross linker path
	// to the compiler via -B ${linker_path}. This is because for gcc, it can
	// find the right linker via searching its internal paths. Clang does not have
	// such feature, and it falls back to $PATH search only. However, the path of
	// ${SDK_LOCATION}/bin is not necessarily in the ${PATH}. To fix this, we
	// provide the directory that contains the cross linker wrapper to clang.
	// Outside chroot, it is the top bin directory form the sdk tarball.
	return filepath.Join(rootPath, "bin")
}

func hasAtLeastOnePrefix(s string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(s, prefix) {
			return true
		}
	}
	return false
}
