# Copyright (c) 2022 Fyde Innovations and the openFyde Authors.
# Distributed under the license specified in the root directory of this project.

# Copyright 2020 The Chromium OS Authors. All rights reserved.
# Distributed under the terms of the GNU General Public License v2

EAPI=7

DESCRIPTION="Virtual for presence of a Vulkan ICD (Installable Client Driver)"
SRC_URI=""

SLOT="0"
KEYWORDS="*"
IUSE=""

# Include the Vulkan loader as a dependency because an ICD is typically not
# useful without it. A Vulkan client can use an ICD directly, without a loader,
# only if (a) the ICD statically exposes its Vulkan entrypoints (most ICDs
# don't) or (b) the client itself implements the loader interface.
RDEPEND="
	media-libs/vulkan-loader
	media-libs/mesa-amd[vulkan]
"
DEPEND=""
