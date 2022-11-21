#!/bin/bash
if [ `uname` = Linux ]; then
  export CC=o64-clang
  export CXX=o64-clang++
else
  export CC=clang
  export CXX=clang++
fi

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
GOOS=darwin GOARCH=amd64 P=osx64 LF="-Wl,-all_load" LD="-pthread -lluajit -lm -ldl -lstdc++" T="vxserver" "${DIR}"/build.sh
