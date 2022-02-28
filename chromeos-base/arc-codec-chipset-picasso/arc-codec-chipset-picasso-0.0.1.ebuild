# Copyright 2020 The Chromium OS Authors. All rights reserved.
# Distributed under the terms of the GNU General Public License v2

EAPI=6

inherit cros-constants

DESCRIPTION="Install codec configuration for ARC++"

LICENSE="BSD-Google"
SLOT="0"
KEYWORDS="*"
IUSE=""
S="${WORKDIR}"

RDEPEND="!chromeos-base/arc-codec-software"

src_install() {
	insinto "${ARC_VENDOR_DIR}/etc/"
	doins "${FILESDIR}"/*
}
