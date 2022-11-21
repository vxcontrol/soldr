#!/bin/bash
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
GOOS=linux GOARCH=amd64 P=linux64 LF="-Wl,--whole-archive" LD="-Wl,--no-whole-archive -pthread -lluajit -lm -ldl -lstdc++" T="vxagent" "${DIR}"/build.sh
