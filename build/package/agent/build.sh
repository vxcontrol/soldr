#!/bin/bash
set -e

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
ROOT_DIR=$(realpath "$DIR/../../..")

BUILD_ARTIFACTS_DIR="$ROOT_DIR/build/artifacts/agent"

[ -n "${PACKAGE_VER+set}" ] || PACKAGE_VER=$(git describe --always `git rev-list --tags --max-count=1`)
[ -n "${PACKAGE_REV+set}" ] || PACKAGE_REV=$(git rev-parse --short HEAD)

export VERSION_STRING="$PACKAGE_VER"
[ "$PACKAGE_REV" ] && VERSION_STRING="$VERSION_STRING-$PACKAGE_REV"
mkdir -p "$BUILD_ARTIFACTS_DIR"
echo $VERSION_STRING > "$BUILD_ARTIFACTS_DIR/version"

[ `uname` = "Darwin" ] && BASE64="base64" || BASE64="base64 -w 0"
[ -n "${STRIP+set}" ] || STRIP="strip"

# go get
# for debugging -gcflags="all=-N -l"
PROTOCOL_VERSION="${API_VERSION:-v1}"
OUT_BIN="${OUT_BIN:-"$ROOT_DIR/build/bin/$T"}"

IAC_CERT=$(cat $ROOT_DIR/security/certs/agent/iac.cert | eval "$BASE64" )
IAC_KEY=$(cat $ROOT_DIR/security/certs/agent/iac.key | eval "$BASE64" )
VXCA_CERT=$(cat $ROOT_DIR/security/certs/agent/vxca.cert | eval "$BASE64" )
IAC_DECODE_KEY="*"
IAC_KEY_DECODE_KEY="+"
VXCA_DECODE_KEY=","

XOREncryptCerts(){
    Asc() { printf '%d' "'$1"; }

    XOR() {
        local s=$1
        local key=$(printf '%d' "'$2'")
        local data_out

        for (( ptr=0; ptr < ${#s}; ptr++ )); do
            c=$( Asc "${s:$ptr:1}" )
            res=$(( c ^ key ))
            data_out+=$(printf '%02x' "$res")
        done

        printf '%s' "$data_out"
    }

    IAC_CERT=$(XOR "$IAC_CERT" "$IAC_DECODE_KEY")
    IAC_KEY=$(XOR $IAC_KEY $IAC_KEY_DECODE_KEY)
    VXCA_CERT=$(XOR $VXCA_CERT $VXCA_DECODE_KEY)
}

XOREncryptCerts

CGO_ENABLED=1 go build -gcflags="all=-N -l" -ldflags "\
    -X soldr/internal/app/agent/config.PackageVer=$PACKAGE_VER \
    -X soldr/internal/app/agent/mmodule.protocolVersion=$PROTOCOL_VERSION \
    -X soldr/internal/app/agent/config.PackageRev=$PACKAGE_REV \
    -X soldr/internal/hardening/luavm/certs/provider.iac=$IAC_CERT \
    -X soldr/internal/hardening/luavm/certs/provider.iacKey=$IAC_KEY \
    -X soldr/internal/hardening/luavm/certs/provider.vxca=$VXCA_CERT \
    -X soldr/internal/hardening/luavm/certs/provider.iacDecodeKey=$IAC_DECODE_KEY \
    -X soldr/internal/hardening/luavm/certs/provider.iacKeyDecodeKey=$IAC_KEY_DECODE_KEY \
    -X soldr/internal/hardening/luavm/certs/provider.vxcaDecodeKey=$VXCA_DECODE_KEY \
    -L $ROOT_DIR/assets/lib/$P -extldflags '$LF $ROOT_DIR/assets/lib/$P/libluab.a $LD'" -o "$OUT_BIN" "$ROOT_DIR"/cmd/agent

[[ -z "${PACKAGE_REV}" && "${GOOS}" != "darwin" ]] && $STRIP "$OUT_BIN"

echo "Calculating ABH for the binary"

ABH=$(sha256sum $OUT_BIN | awk '{print $1}')
JQ_CMD=".v1.agents += {\"$VERSION_STRING/$GOOS/$GOARCH\": \"$ABH\"}"
ABH_FILE="$ROOT_DIR/security/vconf/hardening/abh.json"
if [ -f $ABH_FILE ]; then
    cat "$ABH_FILE" | jq -M --indent 2 "$JQ_CMD" > "$ABH_FILE.tmp" && mv "$ABH_FILE.tmp" "$ABH_FILE"
else
    cat <<EOT >> $ABH_FILE
{
  "v1": {
    "agents": {
      "$VERSION_STRING/$GOOS/$GOARCH": "$ABH"
    },
    "browsers": {},
    "externals": {}
  }
}
EOT
fi
