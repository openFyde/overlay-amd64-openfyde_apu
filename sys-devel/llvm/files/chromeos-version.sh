#!/bin/bash -eu
# Copyright 2023 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

# This script determines the automatic naming scheme for LLVM-related
# ebuilds. e.g. '17.0_pre511803'. It's invoked by
# portage_util.Ebuild.GetVersion through PUPr or cros_uprev as:
#
# bash -x \
#  <full path to package>/files/chromeos-version.sh \
#  <full path to CROS_WORKON_LOCALNAME value>

set -o pipefail

LLVM_PROJECT_DIR="$1"

CROS_VERSION_FILE="${LLVM_PROJECT_DIR}/cros/llvm-rev"
if [[ -e "${CROS_VERSION_FILE}" ]]; then
  LLVM_SVN_REV="$(<"${CROS_VERSION_FILE}")"
else
  CUR_FILES_DIR="$(dirname "$(readlink -f "$0")")"
  GIT_LLVM_REV="${CUR_FILES_DIR}/patch_manager/git_llvm_rev.py"
  HEAD_SHA="$(git -C "${LLVM_PROJECT_DIR}" rev-parse HEAD)"
  LLVM_SVN_REV="$("${GIT_LLVM_REV}"  --llvm_dir "${LLVM_PROJECT_DIR}" \
      --sha "${HEAD_SHA}" | cut -d 'r' -f 2)"
fi

get_cmake_version() {
  local var="$1"
  local cmakefile="$2"
  grep -oE "set\(\s*${var}\s+[0-9]+\s*\)" "${cmakefile}" \
    | sed -E "s/.*${var}\s+([0-9]+).*/\1/g"
}

LLVM_VERSION_FILES=(
  "${LLVM_PROJECT_DIR}/cmake/Modules/LLVMVersion.cmake"
  # Old fallback for finding the revision numbers (see b/335431563). In LLVM-18
  # and prior, the version numbers used to be stored in llvm/CMakeLists.txt.
  "${LLVM_PROJECT_DIR}/llvm/CMakeLists.txt"
)

for version_file in "${LLVM_VERSION_FILES[@]}"; do
  if [[ ! -f "${version_file}" ]]; then
    continue
  fi
  LLVM_MAJOR="$(get_cmake_version 'LLVM_VERSION_MAJOR' "${version_file}")"
  LLVM_MINOR="$(get_cmake_version 'LLVM_VERSION_MINOR' "${version_file}")"
  if [[ -n "${LLVM_MAJOR}" ]]; then
    break
  fi
done

if [[ -z "${LLVM_MAJOR}" ]]; then
  >&2 echo "No LLVM_VERSION_MAJOR detected for LLVM. Has version declaration moved?"
  exit 1
fi
if [[ -z "${LLVM_MINOR}" ]]; then
  >&2 echo "No LLVM_VERSION_MINOR detected for LLVM. Has version declaration moved?"
  exit 1
fi

echo "${LLVM_MAJOR}.${LLVM_MINOR}_pre${LLVM_SVN_REV}"
