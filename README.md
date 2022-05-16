# overlay-amd64-openfyde_apu

Copyright (c) 2022 Fyde Innovations and the openFyde Authors.

Distributed under the license specified in the root directory of this project.

This repository is part of [openfyde](https://github.com/openFyde/) and can be used to set up a BOARD named `overlay-amd64-openfyde_apu`.

This repository contains the following packages:

| Packages                   | Description                                                      | Reference                                                                                                                           |
|----------------------------|------------------------------------------------------------------|-------------------------------------------------------------------------------------------------------------------------------------|
| chromeos-base/device-appid | Setup device appid                                               |                                                                                                                                     |
| virtual/fydeos-board-spec  | Virtual for fydeos board setup                                   |                                                                                                                                     |
| virtual/opengles           | Virtual for OpenGLES implementations                             | [chromiumos-overlay](https://chromium.googlesource.com/chromiumos/overlays/chromiumos-overlay/+/refs/heads/main/virtual/opengles)   |
| virtual/vulkan-icd         | Virtual for presence of a Vulkan ICD (Installable Client Driver) | [chromiumos-overlay](https://chromium.googlesource.com/chromiumos/overlays/chromiumos-overlay/+/refs/heads/main/virtual/vulkan-icd) |
