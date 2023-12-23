#!/bin/bash
set -e

TARGET_DIRECTORY="target"
PRODUCT="vxagent"
OUTPUT_PATH="install_osx"
export VERSION=${VERSION/v/}
export VERSION=${VERSION%-*}


log_info() {
  echo "${LOG_PREFIX}[INFO]" $1
}

log_warn() {
  echo "${LOG_PREFIX}[WARN]" $1
}

log_error() {
  echo "${LOG_PREFIX}[ERROR]" $1
}

deleteInstallationDirectory() {
  log_info "Cleaning $TARGET_DIRECTORY directory."
  rm -rf $TARGET_DIRECTORY

  if [[ $? != 0 ]]; then
    log_error "Failed to clean $TARGET_DIRECTORY directory" $?
    exit 1
  fi
}

createInstallationDirectory() {
  if [ -d ${TARGET_DIRECTORY} ]; then
    deleteInstallationDirectory
  fi
  mkdir -p $TARGET_DIRECTORY

  if [[ $? != 0 ]]; then
    log_error "Failed to create $TARGET_DIRECTORY directory" $?
    exit 1
  fi
}

deleteOutputPath() {
  log_info "Cleaning ${OUTPUT_PATH} directory."
  rm -rf ${OUTPUT_PATH}

  if [[ $? != 0 ]]; then
    log_error "Failed to clean ${OUTPUT_PATH} directory" $?
    exit 1
  fi
}

createOutputPath() {
  if [ -d ${OUTPUT_PATH} ]; then
    deleteOutputPath
  fi
  mkdir -p ${OUTPUT_PATH}

    if [[ $? != 0 ]]; then
    log_error "Failed to create $TARGET_DIRECTORY directory" $?
    exit 1
  fi

}

copyDarwinDirectory() {
  createInstallationDirectory
  createOutputPath
  cp -r darwin ${TARGET_DIRECTORY}/
  chmod -R 755 ${TARGET_DIRECTORY}/darwin
}

copyBuildDirectory() {
  sed -i '' -e 's/__VERSION__/'${VERSION}'/g;s/__PRODUCT__/'${PRODUCT}'/g' ${TARGET_DIRECTORY}/darwin/scripts/postinstall
  sed -i '' -e 's/__VERSION__/'${VERSION}'/g;s/__PRODUCT__/'${PRODUCT}'/g' ${TARGET_DIRECTORY}/darwin/Distribution
  sed -i '' -e 's/__VERSION__/'${VERSION}'/g;s/__PRODUCT__/'${PRODUCT}'/g' ${TARGET_DIRECTORY}/darwin/Resources/*.html

  rm -rf ${TARGET_DIRECTORY}/darwinpkg
  mkdir -p ${TARGET_DIRECTORY}/darwinpkg

  #Copy product to /Library/PRODUCT
  mkdir -p ${TARGET_DIRECTORY}/darwinpkg/Library/${PRODUCT}
  log_info "Try signing executable file"
  cp -a _tmp/darwin/amd64/vxagent ${TARGET_DIRECTORY}/darwinpkg/Library/${PRODUCT}
  chmod -R 755 ${TARGET_DIRECTORY}/darwinpkg/Library/${PRODUCT}

  rm -rf ${TARGET_DIRECTORY}/package
  mkdir -p ${TARGET_DIRECTORY}/package
  chmod -R 755 ${TARGET_DIRECTORY}/package

  rm -rf ${TARGET_DIRECTORY}/pkg
  mkdir -p ${TARGET_DIRECTORY}/pkg
  chmod -R 755 ${TARGET_DIRECTORY}/pkg
}

function buildPackage() {
  log_info "Application installer package building started.(1/2)"
  pkgbuild --identifier org.${PRODUCT}.${VERSION} \
    --version ${VERSION} \
    --scripts ${TARGET_DIRECTORY}/darwin/scripts \
    --root ${TARGET_DIRECTORY}/darwinpkg \
    ${TARGET_DIRECTORY}/package/${PRODUCT}.pkg >/dev/null 2>&1
}

function buildProduct() {
  log_info "Application installer product building started.(2/2)"
  productbuild --distribution ${TARGET_DIRECTORY}/darwin/Distribution \
    --resources ${TARGET_DIRECTORY}/darwin/Resources \
    --package-path ${TARGET_DIRECTORY}/package \
    ${TARGET_DIRECTORY}/pkg/$1 >/dev/null 2>&1
}

function createInstaller() {
  log_info "Application installer generation process started.(2 Steps)"
  buildPackage
  buildProduct ${PRODUCT}-${VERSION}_amd64.pkg
  mv ${TARGET_DIRECTORY}/pkg/${PRODUCT}-${VERSION}_amd64.pkg ${OUTPUT_PATH}/${PRODUCT}-${VERSION}_amd64.pkg
  log_info "Application installer generation steps finished."
}

function createUninstaller() {
  cp darwin/Resources/uninstall.sh ${TARGET_DIRECTORY}/darwinpkg/Library/${PRODUCT}
  sed -i '' -e "s/__VERSION__/${VERSION}/g;s/__PRODUCT__/${PRODUCT}/g" "${TARGET_DIRECTORY}/darwinpkg/Library/${PRODUCT}/uninstall.sh"
}


log_info "Installer generating process started."

copyDarwinDirectory
copyBuildDirectory
createUninstaller
createInstaller

log_info "Installer generating process finished"
exit 0
