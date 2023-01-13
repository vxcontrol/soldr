#!/bin/bash
set -e

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
ROOT_DIR=$(realpath "$DIR/../../..")

BUILD_ARTIFACTS_DIR="$ROOT_DIR/build/artifacts/api"

[ -n "${PACKAGE_VER+set}" ] || PACKAGE_VER=$(git describe --always `git rev-list --tags --max-count=1` | awk -F'-' '{ print $1 }')
[ -n "${PACKAGE_REV+set}" ] || PACKAGE_REV=$(git rev-parse --short HEAD)

BUILD_VERSION="${GITHUB_RUN_NUMBER:-0}"
export VERSION_STRING="$PACKAGE_VER.$BUILD_VERSION"
[ "$PACKAGE_REV" ] && VERSION_STRING="$VERSION_STRING-$PACKAGE_REV"
mkdir -p "$BUILD_ARTIFACTS_DIR"
echo $VERSION_STRING > "$BUILD_ARTIFACTS_DIR/version"

BRANCH=$(git rev-parse --abbrev-ref HEAD)
DEVELOP="true"
case "$BRANCH" in
 "main" ) DEVELOP="false";;
 "release-"* ) DEVELOP="false";;
esac

DB_ENCRYPT_KEY=$(<"$ROOT_DIR/pkg/app/api/utils/dbencryptor/sec-store-key.txt")

[ "$DEBUG" = "true" ] && DEBUG_FLAGS=(-gcflags=all="-N -l")
OUT_BIN="${OUT_BIN:-"$ROOT_DIR/build/bin/vxapi"}"

go build "${DEBUG_FLAGS[@]}" -ldflags "
    -X soldr/pkg/version.IsDevelop=$DEVELOP \
    -X soldr/pkg/version.PackageVer=$PACKAGE_VER.$BUILD_VERSION \
    -X soldr/pkg/version.PackageRev=$PACKAGE_REV \
    -X soldr/pkg/app/server/mmodule/hardening/v1/crypto.DBEncryptKey=$DB_ENCRYPT_KEY" \
    -o "$OUT_BIN" "$ROOT_DIR/cmd/api"

ABH=$(sha256sum $OUT_BIN | awk '{print $1}')
JQ_CMD=".v1.browsers += {\"$VERSION_STRING\": \"$ABH\"} | .v1.externals += {\"$VERSION_STRING\": \"$ABH\"} | .v1.aggregates += {\"$VERSION_STRING\": \"$ABH\"}"
ABH_FILE="$ROOT_DIR/security/vconf/hardening/abh.json"
if [ -f $ABH_FILE ]; then
    cat "$ABH_FILE" | jq -M --indent 2 "$JQ_CMD" > "$ABH_FILE.tmp" && mv "$ABH_FILE.tmp" "$ABH_FILE"
else
    cat <<EOT >> $ABH_FILE
{
  "v1": {
    "agents": {},
    "browsers": {
      "$VERSION_STRING": "$ABH"
    },
    "externals": {
      "$VERSION_STRING": "$ABH"
    },
    "aggregates": {
      "$VERSION_STRING": "$ABH"
    }
  }
}
EOT
fi
