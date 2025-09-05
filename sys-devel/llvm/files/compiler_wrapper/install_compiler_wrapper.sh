#!/bin/bash -eu
#
# Copyright 2020 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

# This script rebuilds and installs compiler wrappers

if [[ ! -e /etc/cros_chroot_version ]]; then
  echo "Please run this script inside chroot"
  exit 1
fi

# Use a unique value here, since folks doing wrapper dev _likely_ want builds
# to always be redone.
version_suffix="manually_installed_wrapper_at_unix_$(date +%s.%6N)"
echo "Using toolchain hash: ${version_suffix}"
clang_version="$(clang --version 2>&1 | head -n1)"
case "${clang_version}" in
	*" 9999 "* )
		# If this was cros-workon'ed, we need to hunt inside of the LLVM repo.
		# The version, on ChromeOS branches, should always be at
		# ${llvm_project}/cros/llvm-rev.
		my_dir="$(dirname "$(readlink -m "$0")")"
		llvm_project="${my_dir}/../../../../../llvm-project"
		llvm_revision="$(<"${llvm_project}/cros/llvm-rev")"
		;;

	*"_pre"* )
		# This can't trivially be replaced with `${foo/bar/baz}`, since there's a
		# backref.
		# shellcheck disable=SC2001
		llvm_revision="$(sed 's/.*_pre\([0-9]*\).*/\1/' <<< "${clang_version}")"
		;;

esac

if [[ -z "${llvm_revision:-}" || "${llvm_revision:-}" =~ [^0-9] ]]; then
  echo "Failed to autodetect current LLVM revision" >&2
  exit 1
fi
echo "Autodetected Clang revision: ${llvm_revision}"

cd "$(dirname "$(readlink -m "$0")")"

build_py() {
  ./build.py \
    --version_suffix="${version_suffix}" \
    --llvm_revision="${llvm_revision}" \
    "$@"
}

install_sysroot_wrappers() {
  local hardness="$1"
  local wrapper_prefix="$2"

  local wrapper="${wrapper_prefix}.noccache"
  local wrapper_ccache="${wrapper_prefix}.ccache"

  echo "Updating ${hardness} wrappers: ${wrapper} and ${wrapper_ccache}"
  build_py \
    --config="cros.${hardness}" \
    --use_ccache=false \
    --use_llvm_next=false \
    --output_file="./${wrapper}"
  build_py \
    --config="cros.${hardness}" \
    --use_ccache=true \
    --use_llvm_next=false \
    --output_file="./${wrapper_ccache}"

  # Update clang target wrappers.
  sudo cp "./${wrapper}" "./${wrapper_ccache}" /usr/bin
  echo "Updated clang wrapper /usr/bin/${wrapper}"
  echo "Updated clang wrapper /usr/bin/${wrapper_ccache}"

  if [[ "${hardness}" == "hardened" ]]; then
    # Update hardened GCC target wrappers.
    local GCC gcc_files
    for GCC in cross-x86_64-cros-linux-gnu/gcc cross-armv7a-cros-linux-gnueabihf/gcc cross-aarch64-cros-linux-gnu/gcc; do
      if ! gcc_files="$(equery f "${GCC}")"; then
        if [[ $(equery l "${GCC}" 2>&1 | wc -c) -eq 0 ]]; then
          echo "no ${GCC} package found; skipping" >&2
          continue
        fi
        # Something went wrong, and the equery above probably complained about it.
        return 1
      fi
      echo "Updating non-ccached ${GCC} wrapper."
      sudo cp "./${wrapper}" "$(grep "${wrapper}" <<< "${gcc_files}")"
      grep "${wrapper}" <<< "${gcc_files}"
      echo "Updating ccached ${GCC} wrapper."
      sudo cp "./${wrapper_ccache}" "$(grep "${wrapper_ccache}" <<< "${gcc_files}")"
      grep "${wrapper_ccache}" <<< "${gcc_files}"
    done
  fi
  rm -f "./${wrapper}" "./${wrapper_ccache}"
}

echo "Updated files:"
# Update the host wrapper
build_py \
  --config=cros.host \
  --use_ccache=false \
  --use_llvm_next=false \
  --output_file=./clang_host_wrapper
sudo mv ./clang_host_wrapper /usr/bin/clang_host_wrapper
echo "/usr/bin/clang_host_wrapper"

install_sysroot_wrappers "hardened" "sysroot_wrapper.hardened"
install_sysroot_wrappers "nonhardened" "sysroot_wrapper"
