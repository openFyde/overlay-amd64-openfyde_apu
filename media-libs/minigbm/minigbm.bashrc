cros_pre_src_prepare_openapu() {
  if [ ${PV} == 9999 ]; then
    return
  fi
  eapply ${OPENFYDE_APU_BASHRC_FILESDIR}/001-fix-DRI-FORMAT.patch
}
