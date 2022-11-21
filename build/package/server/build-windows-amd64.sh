#!/bin/bash
if [ `uname` = Linux ]; then
  export CC=x86_64-w64-mingw32-gcc
  export CXX=x86_64-w64-mingw32-g++
fi

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
GOOS=windows GOARCH=amd64 P=mingw64 LF="-Wl,--export-all-symbols -Wl,--whole-archive" LD="-Wl,--no-whole-archive -lcrypt32 -lgdi32 -lmsimg32 -lopengl32 -lwinmm -lws2_32 -lole32 -lpsapi -lmpr -lluajit -lstdc++" T="vxserver.exe" "${DIR}"/build.sh
