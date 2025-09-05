Copyright 2023 The ChromiumOS Authors
Use of this source code is governed by a BSD-style license that can be
found in the LICENSE file.

### What
Toolchain utils compiler wrapper sources.

Please note that there's a regular syncing operation between
`chromiumos-overlay/sys-devel/llvm/files/compiler_wrapper` and
`toolchain-utils/compiler_wrapper`. This sync is one way (from
chromiumos-overlay to `toolchain-utils`). Syncing in this way helps the Android
toolchain keep up-to-date with our wrapper easily, as they're a downstream
consumer of it. For this reason, **please be sure to land all actual changes in
chromeos-overlay**.

### Build + Run Tests
1. Install the wrapper locally in chroot (builds as well)
```
(chroot) ./install_compiler_wrapper.sh
```

#### Running a manual test
Test a manual build command with `-print-cmdline`
```
(chroot) x86_64-cros-linux-gnu-clang++ -o test_exec -f<some_flag_to_add>='some_value' -print-cmdline test.cc
```
-  `test.cc` doesn't actually have to exist.
-  The command above will output the additional build flags that are added in by the wrapper.

#### Testing your changes
1. Add tests to your wrapper changes
1. Run all the tests via:
```
go test -vet=all
```

### Build Only
This is handy if you just want to test that the build works.

Build the wrapper:
```
./build.py --config=<config name> --use_ccache=<bool> \
  --use_llvm_next=<bool> --output_file=<file>
  ```
