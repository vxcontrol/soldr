#!/bin/bash
set -e

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
ROOT_DIR=$(realpath "$DIR/../../..")

BUILD_ARTIFACTS_DIR="$ROOT_DIR/build/artifacts/server"

cd "$ROOT_DIR"
[ -n "${PACKAGE_VER+set}" ] || PACKAGE_VER=$(git describe --always `git rev-list --tags --max-count=1` | awk -F'-' '{ print $1 }')
[ -n "${PACKAGE_REV+set}" ] || PACKAGE_REV=$(git rev-parse --short HEAD)
cd -

[ -n "${LATEST_API_VERSION}" ] || LATEST_API_VERSION="v1"
hasSpacesRE=" |'"
if [[ "$LATEST_API_VERSION" =~ $hasSpacesRE ]]; then
	echo "The passed LATEST_API_VERSION envvar \"$LATEST_API_VERSION\" contains spaces which is forbidden. Aborting"
	exit 1
fi

BUILD_VERSION="${GITHUB_RUN_NUMBER:-0}"
export VERSION_STRING="$PACKAGE_VER.$BUILD_VERSION"
[ "$PACKAGE_REV" ] && VERSION_STRING="$VERSION_STRING-$PACKAGE_REV"
mkdir -p "$BUILD_ARTIFACTS_DIR"
echo $VERSION_STRING > "$BUILD_ARTIFACTS_DIR/version"

#TODO generate file if file absent in makefile
DB_ENCRYPT_KEY=$(<"$ROOT_DIR/pkg/app/api/utils/dbencryptor/sec-store-key.txt")

export BASE_PREFIX="$ROOT_DIR/assets/lib"
[ "$DEBUG" = "true" ] && DEBUG_FLAGS=(-gcflags=all="-N -l")
CGO_ENABLED=1 go build "${DEBUG_FLAGS[@]}" -ldflags "
    -X soldr/main.PackageVer=$PACKAGE_VER.$BUILD_VERSION \
    -X soldr/main.PackageRev=$PACKAGE_REV \
    -X soldr/pkg/app/server/config.latestAPIVersion=$LATEST_API_VERSION \
    -X soldr/pkg/app/server/mmodule/hardening/v1/crypto.DBEncryptKey=$DB_ENCRYPT_KEY \
    -L $BASE_PREFIX/$P -extldflags '$LF $BASE_PREFIX/$P/libluab.a $LD'" -o "$DIR/../../bin/$T" $DIR/../../../cmd/server
