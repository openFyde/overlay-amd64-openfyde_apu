# Copyright 1999-2019 Gentoo Foundation
# Distributed under the terms of the GNU General Public License v2

EAPI=7

CROS_WORKON_COMMIT="0aa9930caa05507e025f764560fedb0ec3081218"
CROS_WORKON_TREE="4410d8c151d9b163f8929bf66d675ff72415e807"
CROS_WORKON_PROJECT="chromiumos/third_party/mesa"
CROS_WORKON_LOCALNAME="mesa-amd"
CROS_WORKON_EGIT_BRANCH="cros/chromeos-radv"

inherit flag-o-matic meson cros-workon

DESCRIPTION="The Mesa 3D Graphics Library"
HOMEPAGE="http://mesa3d.org/"

LICENSE="MIT"
SLOT="0"
KEYWORDS="~*"

IUSE="debug libglvnd vulkan +zstd"

RDEPEND="
	libglvnd? ( media-libs/libglvnd )
	!libglvnd? ( !media-libs/libglvnd )
	virtual/libelf
	dev-libs/expat
	x11-libs/libdrm
	sys-libs/zlib:=
	zstd? ( app-arch/zstd )
	!media-libs/mesa
	vulkan? ( !media-libs/mesa-radv )
"

DEPEND="${RDEPEND}
	x11-libs/libva
	sys-devel/llvm:15
"

BDEPEND="
	sys-devel/bison
	sys-devel/flex
	virtual/pkgconfig
"

src_configure() {
	cros_optimize_package_for_speed

	export LLVM_CONFIG=${SYSROOT}/usr/lib/llvm/bin/llvm-config-host

	emesonargs+=(
		-Dexecmem=false
		$(meson_use libglvnd glvnd)
		-Dshader-cache-default=false
		-Dglx=disabled
		-Dllvm=enabled
		-Dshared-llvm=disabled
		-Dplatforms=
		-Degl=enabled
		-Dgbm=disabled
		-Dgles1=disabled
		-Dgles2=enabled
		$(meson_feature zstd)
		-Dgallium-drivers=radeonsi
		-Dvulkan-drivers=$(usex vulkan amd '')
		--buildtype $(usex debug debug release)
		-Dgallium-va=enabled
		-Dva-libs-path="/usr/$(get_libdir)/va/drivers"
		-Dvideo-codecs="h264dec,h264enc,h265dec,h265enc,vc1dec"
	)

	meson_src_configure
}

src_install() {
	meson_src_install
  dosym libgallium-${PV}.so /usr/$(get_libdir)/libgallium_dri.so
  dosym ../libgallium-${PV}.so /usr/$(get_libdir)/dri/radeonsi_dri.so
	# Keep the dri header for minigbm
	rm -v -rf "${ED}"/usr/include/GL/*.h
	rm -v -rf "${ED}"/usr/include/{EGL,GLES2,GLES3,KHR}

	# Set driconf option to disable PROTECTED bit check
	insinto "/etc/"
	doins "${FILESDIR}"/drirc
}
