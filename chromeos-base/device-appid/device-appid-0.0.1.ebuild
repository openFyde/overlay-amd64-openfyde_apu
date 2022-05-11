# Copyright (c) 2022 Fyde Innovations and the openFyde Authors.
# Distributed under the license specified in the root directory of this project.

EAPI="5"

inherit appid2

DESCRIPTION="empty project"
HOMEPAGE="http://fydeos.com"

LICENSE="BSD-Google"
SLOT="0"
KEYWORDS="*"
IUSE=""

RDEPEND=""

DEPEND="${RDEPEND}"

S="${WORKDIR}"

src_install() {
	doappid "{28206D4D-E45B-4374-BA4A-58AE2723E600}" "CHROMEBOOK" "{C5ABC7D8-B432-4B22-80C7-C0D296A642B6}"
}
