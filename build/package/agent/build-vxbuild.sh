#!/bin/bash

set -e

SCRIPT_DIR="$(dirname "$0")"

docker run --rm -it \
        -u $(id -u):$(id -g) \
        -v $(realpath "$SCRIPT_DIR/../../../../"):/go/src/ \
        -v $(go env GOMODCACHE):/go/pkg/mod \
        -e GOCACHE=/tmp \
        --workdir=/go/src/soldr \
        vxcontrol/vxbuild-cross:latest /bin/bash -c "build/package/agent/build-$(go env GOOS)-$(go env GOARCH).sh"
