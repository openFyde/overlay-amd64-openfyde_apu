#2 Copyright 1999-2023 Gentoo Authors
# Distributed under the terms of the GNU General Public License v2

EAPI=7

: ${CMAKE_MAKEFILE_GENERATOR:=ninja}
PYTHON_COMPAT=( python3_11 )

inherit cmake flag-o-matic multilib-minimal \
	multiprocessing pax-utils python-any-r1 toolchain-funcs arc-build

DESCRIPTION="Low Level Virtual Machine"
HOMEPAGE="https://llvm.org/"
SRC_URI="https://github.com/llvm/llvm-project/releases/download/llvmorg-${PV/_/-}/llvm-project-${PV/_/-}.src.tar.xz
	!doc? ( https://dev.gentoo.org/~mgorny/dist/llvm/llvm-18.1.0-manpages.tar.bz2 )"

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
IUSE="debug doc exegesis libedit libffi ncurses test xar xml
	kernel_Darwin ${ALL_LLVM_TARGETS[*]}"
RESTRICT="!test? ( test )"

# There are no file collisions between these versions but having :0
# installed means llvm-config there will take precedence.
RDEPEND="!sys-devel/arc-llvm:0"
# Remove previous version of llvm to avoid file collisions, since all slots end
# up in the same install directory.
RDEPEND="${RDEPEND}
	!<sys-devel/arc-llvm-${SLOT}"

REQUIRED_USE="|| ( ${ALL_LLVM_TARGETS[*]} )"

S=${WORKDIR}/llvm-project-${PV}.src/llvm

HOST_DIR="${WORKDIR}/${PF}-${CBUILD}"

# least intrusive of all
CMAKE_BUILD_TYPE=RelWithDebInfo

src_prepare() {
	# Fix llvm-config for shared linking and sane flags
	# https://bugs.gentoo.org/show_bug.cgi?id=565358
	# eapply "${FILESDIR}"/9999/0007-llvm-config-Clean-up-exported-values-update-for-shar.patch

	# Don't install static libraries when not requested
	eapply "${FILESDIR}/arc-llvm-9.0.0-no-static-libraries.patch"

	if [[ ${ARC_LLVM_VERSION} == 6* ]]; then
		eapply "${FILESDIR}/0001-CycleInfo-work-around-a-compiler-issue.patch"
	fi

	# disable use of SDK on OSX, bug #568758
	sed -i -e 's/xcrun/false/' utils/lit/lit/util.py || die

	# User patches + QA
	eapply_user

	cmake_src_prepare
}

build_host_tool() {
	local tool="$1"
	# Use host toolchain when building for the host.
	local CC=clang
	local CXX=clang++
	local CFLAGS=''
	local CXXFLAGS=''
	local LDFLAGS=''
	mkdir -p "${HOST_DIR}" || die
	cd "${HOST_DIR}" || die
	local libdir=$(get_libdir)
	cmake -DLLVM_LIBDIR_SUFFIX=${libdir#lib} \
		-DLLVM_TARGETS_TO_BUILD="${LLVM_TARGETS// /;}" \
		-DCMAKE_BUILD_TYPE=RelWithDebInfo \
		-G "Unix Makefiles" ${S}
	# Settings for the target and host may differ (e.g. system libs), but we
	# need llvm-config's output to match the target behavior. Copy the
	# config from the target into the host before embedding into the binary.
	if [[ -f "${BUILD_DIR}/${tool}/BuildVariables.inc" ]]; then
		cp "${BUILD_DIR}/${tool}/BuildVariables.inc" \
			"${HOST_DIR}/${tool}"
	fi
	cd "${HOST_DIR}/${tool}" || die
	emake
}

build_host_tblgen() {
	build_host_tool "utils/TableGen"
}

build_host_config() {
	build_host_tool "tools/llvm-config"
	mv "${HOST_DIR}/bin/llvm-config" "${HOST_DIR}/bin/llvm-config-${ABI}"
}

src_configure() {
	arc-build-select-clang
	multilib-minimal_src_configure
}

multilib_src_configure() {
	local ffi_cflags ffi_ldflags
	if use libffi; then
		ffi_cflags=$($(tc-getPKG_CONFIG) --cflags-only-I libffi)
		ffi_ldflags=$($(tc-getPKG_CONFIG) --libs-only-L libffi)
	fi

	# Workaround for b/253514951
	if [[ ${ABI} == x86 ]] && [[ ${ARC_LLVM_VERSION} == 11* ]]; then
		replace-flags "-march=bdver4" "-march=bdver4 -mno-xop"
	fi
	local libdir=$(get_libdir)
	local mycmakeargs=(
		# disable appending VCS revision to the version to improve
		# direct cache hit ratio
		-DLLVM_APPEND_VC_REV=OFF
		-DCMAKE_INSTALL_PREFIX="${ARC_PREFIX}/build"
		-DLLVM_LIBDIR_SUFFIX=${libdir#lib}

		-DBUILD_SHARED_LIBS=OFF

		-DLLVM_TARGETS_TO_BUILD="${LLVM_TARGETS// /;}"
		-DLLVM_BUILD_TESTS=$(usex test)
		-DLLVM_BUILD_TOOLS=OFF
		-DLLVM_BUILD_RUNTIMES=OFF
		-DLLVM_TOOL_LTO_BUILD=OFF

		-DLLVM_ENABLE_FFI=$(usex libffi)
		-DLLVM_ENABLE_LIBEDIT=$(usex libedit)
		-DLLVM_ENABLE_TERMINFO=$(usex ncurses)
		-DLLVM_ENABLE_LIBXML2=$(usex xml)
		-DLLVM_ENABLE_ASSERTIONS=$(usex debug)
		-DLLVM_ENABLE_LIBPFM=$(usex exegesis)
		-DLLVM_ENABLE_EH=OFF
		-DLLVM_ENABLE_RTTI=ON
		-DLLVM_ENABLE_ZLIB=OFF
		-DLLVM_ENABLE_ZSTD=OFF

		-DWITH_POLLY=OFF # TODO

		-DLLVM_HOST_TRIPLE="${CHOST}"

		-DFFI_INCLUDE_DIR="${ffi_cflags#-I}"
		-DFFI_LIBRARY_DIR="${ffi_ldflags#-L}"
		# used only for llvm-objdump tool
		-DHAVE_LIBXAR=$(multilib_native_usex xar 1 0)

		# disable OCaml bindings (now in dev-ml/llvm-ocaml)
		-DOCAMLFIND=NO
	)

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
	fi

	if tc-is-cross-compiler; then
		# Force LLVM_BUILD_TOOLS=ON to ensure build_host_tools builds
		# llvm-config
		build_host_tblgen
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

	build_host_config
}

multilib_src_test() {
	# respect TMPDIR!
	local -x LIT_PRESERVES_TMP=1
	cmake_build check
}

src_install() {
	local LLVM_LDPATHS=()
	multilib-minimal_src_install
}

multilib_src_install() {
	cmake_src_install

	into ${ARC_PREFIX}/build
	newbin "${HOST_DIR}/bin/llvm-config-${ABI}" "llvm-config-host-${ABI}"
}

multilib_src_install_all() {
	local LLVM_CONFIG_HOST="${D}/${ARC_PREFIX}/build/bin/llvm-config-host"
	cat > "${LLVM_CONFIG_HOST}" <<EOF
#!/bin/bash

ABI_BIN="\$(dirname ""\$0"")/llvm-config-host-\${ABI}"

if [[ -e "\${ABI_BIN}" ]]; then
	exec "\${ABI_BIN}" "\$@"
else
	echo "\$0: Unsupported ABI: \${ABI}"
	exit 1
fi
EOF
	chmod a+rx "${LLVM_CONFIG_HOST}" || die
}
