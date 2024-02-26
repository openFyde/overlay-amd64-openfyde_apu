# Copyright (c) 2022 Fyde Innovations and the openFyde Authors.
# Distributed under the license specified in the root directory of this project.

EAPI=7

DESCRIPTION="empty project"
HOMEPAGE="http://fydeos.com"

LICENSE="BSD"
SLOT="0"
KEYWORDS="*"

RDEPEND="
    virtual/openfyde-board-spec
    chromeos-base/device-appid
    chromeos-base/fydeos-power-daemon-go
"

DEPEND="${RDEPEND}"
