#!/bin/bash

set -ex

git status
git config --global --add safe.directory $(pwd)

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
ROOT_DIR=$(realpath "$DIR/../")

CRTS_DIR="${ROOT_DIR}"/security/certs
CERTS_WRITER_DIR="${ROOT_DIR}"/scripts/certs_writer
SBH_GENERATOR_DIR="${ROOT_DIR}"/scripts/sbh_generator
TMP_DIR="${CERTS_WRITER_DIR}/certs/tmp_$(date +%s)"

mkdir -p \
    "${CRTS_DIR}"/server/vxca \
    "${CRTS_DIR}"/server/sc \
    "${CRTS_DIR}"/server/sca \
    "${CRTS_DIR}"/agent \
    "${CRTS_DIR}"/api/aggregate \
    "${CRTS_DIR}"/api/browser \
    "${CRTS_DIR}"/api/external \
    "${TMP_DIR}"

# Generate certificates
DST_DIR="${TMP_DIR}" make -C "${CERTS_WRITER_DIR}" generate

GEN_CERTS_DIR=$(echo "${TMP_DIR}"/*)
GEN_CERTS_SERVERNAME=$(ls "${TMP_DIR}")

mkdir -p "${CRTS_DIR}"/"${GEN_CERTS_SERVERNAME}"

cp "${GEN_CERTS_DIR}"/* "${CRTS_DIR}"/"${GEN_CERTS_SERVERNAME}"/

cp "${GEN_CERTS_DIR}"/vxca.cert "${CRTS_DIR}"/server/vxca/"${GEN_CERTS_SERVERNAME}".cert
cp "${GEN_CERTS_DIR}"/sc.cert "${CRTS_DIR}"/server/sc/"${GEN_CERTS_SERVERNAME}".cert
cp "${GEN_CERTS_DIR}"/sc.key "${CRTS_DIR}"/server/sc/"${GEN_CERTS_SERVERNAME}".key
cp "${GEN_CERTS_DIR}"/ca.cert "${CRTS_DIR}"/server/sca/"${GEN_CERTS_SERVERNAME}".cert
cp "${GEN_CERTS_DIR}"/ca.key "${CRTS_DIR}"/server/sca/"${GEN_CERTS_SERVERNAME}".key
cp "${GEN_CERTS_DIR}"/vxca.cert "${CRTS_DIR}"/server/vxca/"${GEN_CERTS_SERVERNAME}".cert

cp "${GEN_CERTS_DIR}"/vxca.cert "${CRTS_DIR}"/agent/
cp "${GEN_CERTS_DIR}"/iac.cert "${CRTS_DIR}"/agent/
cp "${GEN_CERTS_DIR}"/iac.key "${CRTS_DIR}"/agent/

cp "${GEN_CERTS_DIR}"/vxca.cert "${CRTS_DIR}"/api/
cp "${GEN_CERTS_DIR}"/ca.cert "${CRTS_DIR}"/api/aggregate/
cp "${GEN_CERTS_DIR}"/ca.cert "${CRTS_DIR}"/api/browser/
cp "${GEN_CERTS_DIR}"/ca.cert "${CRTS_DIR}"/api/external/
cp "${GEN_CERTS_DIR}"/ltac_aggregate.cert "${CRTS_DIR}"/api/aggregate/ltac.cert
cp "${GEN_CERTS_DIR}"/ltac_aggregate.key "${CRTS_DIR}"/api/aggregate/ltac.key
cp "${GEN_CERTS_DIR}"/ltac_browser.cert "${CRTS_DIR}"/api/browser/ltac.cert
cp "${GEN_CERTS_DIR}"/ltac_browser.key "${CRTS_DIR}"/api/browser/ltac.key
cp "${GEN_CERTS_DIR}"/ltac_external.cert "${CRTS_DIR}"/api/external/ltac.cert
cp "${GEN_CERTS_DIR}"/ltac_external.key "${CRTS_DIR}"/api/external/ltac.key

# Generate server binary hash (sbh)
SERVER_NAME="${GEN_CERTS_SERVERNAME}" make -C ${SBH_GENERATOR_DIR} generate

rm -rf "${TMP_DIR}"
