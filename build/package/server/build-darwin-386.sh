#!/bin/bash
if [ `uname` = Linux ]; then
  export CC=o32-clang
  export CXX=o32-clang++
else
  export CC=clang
  export CXX=clang++
fi

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
GOOS=darwin GOARCH=386 P=osx32 LF="-Wl,-all_load" LD="-pthread -lluajit -lm -ldl -lstdc++" T="vxserver" "${DIR}"/build.sh
