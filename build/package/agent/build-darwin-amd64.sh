#!/bin/bash
if [ `uname` = Linux ]; then
  export CC=o64-clang
  export CXX=o64-clang++
  export STRIP=x86_64-apple-darwin19-strip
else
  export CC=clang
  export CXX=clang++
fi

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
GOOS=darwin GOARCH=amd64 P=osx64 LF="-Wl,-all_load" LD="-pthread -lluajit -lm -ldl -lstdc++" T="vxagent" "${DIR}"/build.sh
