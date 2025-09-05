#!/bin/bash -eux
#
# Copyright 2021 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

# If your compiler wrapper ends up broken, you can run this script to try to
# restore it to a working version. We can only use artifacts we download from
# gs://, since it's kind of hard to build a working compiler with a broken
# compiler wrapper. ;)

if [[ ! -e "/etc/cros_chroot_version" ]]; then
  echo "Run me inside of the chroot."
  exit 1
fi

packages_to_reemerge=(
  # We want to reemerge the host wrapper...
  sys-devel/llvm
)

gcc_wrappers=(
  cross-x86_64-cros-linux-gnu/gcc
  cross-armv7a-cros-linux-gnueabihf/gcc
  cross-aarch64-cros-linux-gnu/gcc
)

# ...and any existing target wrappers.
for gcc in "${gcc_wrappers[@]}"; do
  # cheap check for whether or not the package in question is already installed
  if ls /var/db/pkg/"${gcc}"-* >& /dev/null; then
    packages_to_reemerge+=( "${gcc}" )
  fi
done

# Ensure that we don't pick up any broken binpkgs for these when we install
# them below.
for pkg in "${packages_to_reemerge[@]}"; do
  sudo rm -f "/var/lib/portage/pkgs/${pkg}"*
done

sudo emerge -j16 -G "${packages_to_reemerge[@]}"
