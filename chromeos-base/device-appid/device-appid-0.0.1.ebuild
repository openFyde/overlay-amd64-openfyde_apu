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
  doappid "{FA774243-0570-484F-A53C-8BB7F4FA8F24}" "CHROMEBOOK" "{CDA4D576-795C-45EF-99F3-107476A295D6}"
}
