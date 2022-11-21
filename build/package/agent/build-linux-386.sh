#!/bin/bash
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
GOOS=linux GOARCH=386 P=linux32 LF="-Wl,--whole-archive" LD="-Wl,--no-whole-archive -pthread -lluajit -lm -ldl -lstdc++" T="vxagent" "${DIR}"/build.sh
