openfyde_apu_stack_bashrc() {
  local cfg cfgd

  cfgd="/mnt/host/source/src/overlays/overlay-amd64-openfyde_apu/${CATEGORY}/${PN}"
  for cfg in ${PN} ${P} ${PF} ; do
    cfg="${cfgd}/${cfg}.bashrc"
    [[ -f ${cfg} ]] && . "${cfg}"
  done

  export OPENFYDE_APU_BASHRC_FILESDIR="${cfgd}/files"
}

openfyde_apu_stack_bashrc
