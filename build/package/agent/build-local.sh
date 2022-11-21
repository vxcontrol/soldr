#!/bin/bash
set -e

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
"${DIR}"/build-$(go env GOOS)-$(go env GOARCH).sh
