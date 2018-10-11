#!/bin/bash
readonly TMPDOT="$(mktemp typediag-XXXXXX)"
typediag --path="${1}" > ${TMPDOT}
dot -Tpng -o "${2}" "${TMPDOT}"
rm "${TMPDOT}"
