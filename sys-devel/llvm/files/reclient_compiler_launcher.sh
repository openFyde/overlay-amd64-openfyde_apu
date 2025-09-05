#!/bin/bash -eu
# Copyright 2024 The ChromiumOS Authors
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

# We'll eventually want to pass in the build orchestrator (Bazel or Portage) and
# choose the appropriate value based on the parent orchestrator. But for now we
# don't have enough RBE workers to support building llvm on the "bazel" worker
# pool so we'll hard-code this to "portage" for now.
build_orchestrator="portage"

# Use the same container image as the one used by chrome. It just works and using the same image
# is good in terms of cache efficiency. The container image was built with this Dockerfile:
# https://chromium.googlesource.com/infra/infra/+/main/rbe/images/siso-chromium/linux/Dockerfile
DOCKER_IMAGE=docker://gcr.io/chops-public-images-prod/rbe/siso-chromium/linux@sha256:26de99218a1a8b527d4840490bcbf1690ee0b55c84316300b60776e6b3a03fe1
WORKER_LABELS=label:orchestrator=${build_orchestrator},label:package_accelerator=reclient,label:core_count=2

exec "${CROS_REMOTEEXEC_REWRAPPER_PATH:?}" \
  -platform="container-image=${DOCKER_IMAGE},dockerChrootPath=.,dockerRuntime=runsc,${WORKER_LABELS}" \
  -exec_strategy="remote_local_fallback" \
  -dial_timeout="10m" \
  -exec_timeout="2m" \
  -reclient_timeout="2m" \
  -exec_root="/" \
  -labels="type=compile,compiler=clang,lang=cpp" \
  -- "$@"
