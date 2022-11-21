#!/bin/bash
if [ `uname` = Linux ]; then
  export CC=x86_64-w64-mingw32-gcc
  export CXX=x86_64-w64-mingw32-g++
  export STRIP=x86_64-w64-mingw32-strip
fi

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
[ `uname` = "Linux" ] && GOVERSIONINFO="${DIR}/tools/goversioninfo" || GOVERSIONINFO="goversioninfo"
[ -n "${PACKAGE_VER+set}" ] || PACKAGE_VER=$(git describe --always `git rev-list --tags --max-count=1`)
SYSO=build/rsrc_windows_amd64.syso
$GOVERSIONINFO -product-version $PACKAGE_VER -64 -o $SYSO
GOOS=windows GOARCH=amd64 P=mingw64 LF="$SYSO -static -Wl,--export-all-symbols -Wl,--whole-archive" LD="-Wl,--no-whole-archive -lcrypt32 -lgdi32 -lmsimg32 -lopengl32 -lwinmm -lws2_32 -lole32 -lpsapi -lmpr -lluajit -lstdc++" T="vxagent.exe" "${DIR}"/build.sh
rm $SYSO
