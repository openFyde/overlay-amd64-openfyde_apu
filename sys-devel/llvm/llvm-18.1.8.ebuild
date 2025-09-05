# Copyright 1999-2023 Gentoo Authors
# Distributed under the terms of the GNU General Public License v2

EAPI=7

: ${CMAKE_MAKEFILE_GENERATOR:=ninja}
PYTHON_COMPAT=( python3_{8..11} )

inherit cmake flag-o-matic multilib-minimal \
	multiprocessing pax-utils python-any-r1 toolchain-funcs

DESCRIPTION="Low Level Virtual Machine"
HOMEPAGE="https://llvm.org/"
SRC_URI="https://github.com/llvm/llvm-project/releases/download/llvmorg-${PV/_/-}/llvm-project-${PV/_/-}.src.tar.xz
	!doc? ( https://dev.gentoo.org/~mgorny/dist/llvm/${PN}-18.1.0-manpages.tar.bz2 )"

# Keep in sync with CMakeLists.txt
ALL_LLVM_TARGETS=( AArch64 AMDGPU ARM BPF Hexagon Lanai Mips MSP430
	NVPTX PowerPC Sparc SystemZ WebAssembly X86 XCore )
ALL_LLVM_TARGETS=( "${ALL_LLVM_TARGETS[@]/#/llvm_targets_}" )

# Additional licenses:
# 1. OpenBSD regex: Henry Spencer's license ('rc' in Gentoo) + BSD.
# 2. ARM backend: LLVM Software Grant by ARM.
# 3. MD5 code: public-domain.
# 4. Tests (not installed):
#  a. gtest: BSD.
#  b. YAML tests: MIT.

LICENSE="LLVM-exception rc BSD public-domain
	llvm_targets_ARM? ( LLVM-Grant )"
SLOT="$(ver_cut 1)"
KEYWORDS="~*"
IUSE="debug doc exegesis gold libedit +libffi ncurses test tools xar xml
	kernel_Darwin ${ALL_LLVM_TARGETS[*]}"

# Skip testing; it's not useful, and may lead to slot conflicts: b/328322177
RESTRICT="test mirror"

RDEPEND="
	sys-libs/zlib:0=
	exegesis? ( dev-libs/libpfm:= )
	gold? ( >=sys-devel/binutils-2.31.1-r4:*[plugins] )
	libedit? ( dev-libs/libedit:0=[${MULTILIB_USEDEP}] )
	libffi? ( >=dev-libs/libffi-3.0.13-r1:0=[${MULTILIB_USEDEP}] )
	ncurses? ( >=sys-libs/ncurses-5.9-r3:0=[${MULTILIB_USEDEP}] )
	xar? ( app-arch/xar )
	xml? ( dev-libs/libxml2:2=[${MULTILIB_USEDEP}] )"
DEPEND="${RDEPEND}
	gold? ( sys-libs/binutils-libs )"
BDEPEND="
	dev-lang/perl
	>=dev-util/cmake-3.16
	sys-devel/gnuconfig
	kernel_Darwin? (
		<sys-libs/libcxx-$(ver_cut 1-3).9999
		>=sys-devel/binutils-apple-5.1
	)
	doc? ( $(python_gen_any_dep '
		dev-python/recommonmark[${PYTHON_USEDEP}]
		dev-python/sphinx[${PYTHON_USEDEP}]
	') )
	libffi? ( virtual/pkgconfig )
	${PYTHON_DEPS}"
# Note: Gentoo supports multiple SLOTs of llvm, but we don't currently for the board.
# Force older versions to be removed first.
RDEPEND="${RDEPEND}
	!<sys-devel/llvm-${SLOT}"
PDEPEND="gold? ( sys-devel/llvmgold )"

REQUIRED_USE="|| ( ${ALL_LLVM_TARGETS[*]} )"

S=${WORKDIR}/llvm-project-${PV}.src/llvm

HOST_DIR="${WORKDIR}/${PF}-${CBUILD}"

# least intrusive of all
CMAKE_BUILD_TYPE=RelWithDebInfo

python_check_deps() {
	use doc || return 0

	has_version "dev-python/recommonmark[${PYTHON_USEDEP}]" &&
	has_version "dev-python/sphinx[${PYTHON_USEDEP}]"
}

src_prepare() {
	# Fix llvm-config for shared linking and sane flags
	# https://bugs.gentoo.org/show_bug.cgi?id=565358
	#eapply "${FILESDIR}"/9999/0007-llvm-config-Clean-up-exported-values-update-for-shar.patch

	# disable use of SDK on OSX, bug #568758
	sed -i -e 's/xcrun/false/' utils/lit/lit/util.py || die

	# User patches + QA
	eapply_user

	cmake_src_prepare
}

# Is LLVM being linked against libc++?
is_libcxx_linked() {
	local code='#include <ciso646>
#if defined(_LIBCPP_VERSION)
	HAVE_LIBCXX
#endif
'
	local out=$($(tc-getCXX) ${CXXFLAGS} ${CPPFLAGS} -x c++ -E -P - <<<"${code}") || return 1

	[[ ${out} == *HAVE_LIBCXX* ]]
}

build_host_tools() {
	# Use host toolchain when building for the host.
	local CC=clang
	local CXX=clang++
	local CFLAGS=''
	local CXXFLAGS=''
	local LDFLAGS=''
	mkdir -p "${HOST_DIR}" || die
	cd "${HOST_DIR}" || die
	cmake "${@}" -G "Unix Makefiles" "${S}" || die
	cd "${HOST_DIR}/utils/TableGen" || die
	emake
	cd "${HOST_DIR}/tools/llvm-config" || die
	emake
}

multilib_src_configure() {
	local ffi_cflags ffi_ldflags
	if use libffi; then
		ffi_cflags=$($(tc-getPKG_CONFIG) --cflags-only-I libffi)
		ffi_ldflags=$($(tc-getPKG_CONFIG) --libs-only-L libffi)
	fi

	local libdir=$(get_libdir)
	local mycmakeargs=(
		# disable appending VCS revision to the version to improve
		# direct cache hit ratio
		-DLLVM_APPEND_VC_REV=OFF
		-DCMAKE_INSTALL_PREFIX="${EPREFIX}/usr/lib/llvm"
		-DLLVM_LIBDIR_SUFFIX=${libdir#lib}

		-DBUILD_SHARED_LIBS=OFF

		-DLLVM_TARGETS_TO_BUILD="${LLVM_TARGETS// /;}"
		-DLLVM_BUILD_TESTS=$(usex test)
		-DLLVM_BUILD_TOOLS=$(usex tools)

		-DLLVM_ENABLE_FFI=$(usex libffi)
		-DLLVM_ENABLE_LIBEDIT=$(usex libedit)
		-DLLVM_ENABLE_TERMINFO=$(usex ncurses)
		-DLLVM_ENABLE_LIBXML2=$(usex xml)
		-DLLVM_ENABLE_ASSERTIONS=$(usex debug)
		-DLLVM_ENABLE_LIBPFM=$(usex exegesis)
		-DLLVM_ENABLE_EH=ON
		-DLLVM_ENABLE_RTTI=ON

		-DWITH_POLLY=OFF # TODO

		-DLLVM_HOST_TRIPLE="${CHOST}"

		-DFFI_INCLUDE_DIR="${ffi_cflags#-I}"
		-DFFI_LIBRARY_DIR="${ffi_ldflags#-L}"
		# used only for llvm-objdump tool
		-DHAVE_LIBXAR=$(multilib_native_usex xar 1 0)

		# disable OCaml bindings (now in dev-ml/llvm-ocaml)
		-DOCAMLFIND=NO
		-DCMAKE_BUILD_TYPE=RelWithDebInfo
	)

	if is_libcxx_linked; then
		# Smart hack: alter version suffix -> SOVERSION when linking
		# against libc++. This way we won't end up mixing LLVM libc++
		# libraries with libstdc++ clang, and the other way around.
		mycmakeargs+=(
			-DLLVM_VERSION_SUFFIX="libcxx"
		)
	fi

#	Note: go bindings have no CMake rules at the moment
#	but let's kill the check in case they are introduced
#	if ! multilib_is_native_abi || ! use go; then
		mycmakeargs+=(
			-DGO_EXECUTABLE=GO_EXECUTABLE-NOTFOUND
		)
#	fi

	use test && mycmakeargs+=(
		-DLLVM_LIT_ARGS="-vv;-j;${LIT_JOBS:-$(makeopts_jobs "${MAKEOPTS}" "$(get_nproc)")}"
	)

	if multilib_is_native_abi; then
		mycmakeargs+=(
			-DLLVM_BUILD_DOCS=$(usex doc)
			-DLLVM_ENABLE_OCAMLDOC=OFF
			-DLLVM_ENABLE_SPHINX=$(usex doc)
			-DLLVM_ENABLE_DOXYGEN=OFF
			-DLLVM_INSTALL_UTILS=ON
		)
		use doc && mycmakeargs+=(
			-DCMAKE_INSTALL_MANDIR="${EPREFIX}/usr/lib/llvm/share/man"
			-DLLVM_INSTALL_SPHINX_HTML_DIR="${EPREFIX}/usr/share/doc/${PF}/html"
			-DSPHINX_WARNINGS_AS_ERRORS=OFF
		)
		use gold && mycmakeargs+=(
			-DLLVM_BINUTILS_INCDIR="${EPREFIX}"/usr/include
		)
	fi

	if tc-is-cross-compiler; then
		# Force LLVM_BUILD_TOOLS=ON to ensure build_host_tools builds
		# llvm-config
		build_host_tools "${mycmakeargs[@]}" -DLLVM_BUILD_TOOLS=ON
		# die early if the build tools are not installed
		[[ -x "${HOST_DIR}/bin/llvm-tblgen" ]] \
			|| die "${HOST_DIR}/bin/llvm-tblgen not found or usable"
		mycmakeargs+=(
			-DCMAKE_CROSSCOMPILING=ON
			-DLLVM_TABLEGEN="${HOST_DIR}/bin/llvm-tblgen"
		)
	fi

	# workaround BMI bug in gcc-7 (fixed in 7.4)
	# https://bugs.gentoo.org/649880
	# apply only to x86, https://bugs.gentoo.org/650506
	if tc-is-gcc && [[ ${MULTILIB_ABI_FLAG} == abi_x86* ]] &&
			[[ $(gcc-major-version) -eq 7 && $(gcc-minor-version) -lt 4 ]]
	then
		local CFLAGS="${CFLAGS} -mno-bmi"
		local CXXFLAGS="${CXXFLAGS} -mno-bmi"
	fi

	# LLVM_ENABLE_ASSERTIONS=NO does not guarantee this for us, #614844
	use debug || local -x CPPFLAGS="${CPPFLAGS} -DNDEBUG"
	cmake_src_configure
}

multilib_src_compile() {
	cmake_src_compile

	pax-mark m "${BUILD_DIR}"/bin/llvm-rtdyld
	pax-mark m "${BUILD_DIR}"/bin/lli
	pax-mark m "${BUILD_DIR}"/bin/lli-child-target

	if use test; then
		pax-mark m "${BUILD_DIR}"/unittests/ExecutionEngine/Orc/OrcJITTests
		pax-mark m "${BUILD_DIR}"/unittests/ExecutionEngine/MCJIT/MCJITTests
		pax-mark m "${BUILD_DIR}"/unittests/Support/SupportTests
	fi
}

src_install() {
	local MULTILIB_CHOST_TOOLS=()

	if use tools; then
		MULTILIB_CHOST_TOOLS+=(
			/usr/lib/llvm/bin/llvm-config
		)
	fi

	local MULTILIB_WRAPPED_HEADERS=(
		/usr/include/llvm/Config/llvm-config.h
	)

	local LLVM_LDPATHS=()
	multilib-minimal_src_install

	# move wrapped headers back
	mv "${ED%/}"/usr/include "${ED%/}"/usr/lib/llvm/include || die
	# llvm-config --includedir and --libdir print output based on where
	# llvm-config itself is located. So to get an includedir of
	# .../usr/lib/llvm/include, llvm-config needs to be in
	# .../usr/lib/llvm/bin.
	mv "${HOST_DIR}"/bin/llvm-config "${ED%/}"/usr/lib/llvm/bin/llvm-config-host || die
}

multilib_src_install() {
	cmake_src_install

	# move headers to /usr/include for wrapping
	rm -rf "${ED%/}"/usr/include || die
	mv "${ED%/}"/usr/lib/llvm/include "${ED%/}"/usr/include || die

	LLVM_LDPATHS+=( "${EPREFIX}/usr/lib/llvm/$(get_libdir)" )
}

multilib_src_install_all() {
	local revord=$(( 9999 - ${SLOT} ))
	newenvd - "10llvm-${revord}" <<-_EOF_
		PATH="${EPREFIX}/usr/lib/llvm/bin"
		# we need to duplicate it in ROOTPATH for Portage to respect...
		ROOTPATH="${EPREFIX}/usr/lib/llvm/bin"
		MANPATH="${EPREFIX}/usr/lib/llvm/share/man"
		LDPATH="$( IFS=:; echo "${LLVM_LDPATHS[*]}" )"
	_EOF_

	# install pre-generated manpages
	if ! use doc; then
		# (doman does not support custom paths)
		insinto "/usr/lib/llvm/share/man/man1"
		doins "${WORKDIR}/${PN}-18.1.0-manpages/llvm"/*.1
	fi

	docompress "/usr/lib/llvm/share/man"
}
