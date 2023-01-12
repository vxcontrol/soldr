#!/bin/bash
if [ `uname` = Linux ]; then
  export CC=i686-w64-mingw32-gcc
  export CXX=i686-w64-mingw32-g++
  export STRIP=i686-w64-mingw32-strip
fi

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
chmod +x "${DIR}/tools/goversioninfo"
[ `uname` = "Linux" ] && GOVERSIONINFO="${DIR}/tools/goversioninfo" || GOVERSIONINFO="goversioninfo"
[ -n "${PACKAGE_VER+set}" ] || PACKAGE_VER=$(git describe --always `git rev-list --tags --max-count=1` | awk -F'-' '{ print $1 }')
SYSO="${DIR}/rsrc_windows_386.syso"
$GOVERSIONINFO -product-version $PACKAGE_VER -o $SYSO -icon ${DIR}/images/app.ico ${DIR}/versioninfo.json
GOOS=windows GOARCH=386 P=mingw32 LF="$SYSO -static -Wl,--export-all-symbols -Wl,--whole-archive" LD="-Wl,--no-whole-archive -lcrypt32 -lgdi32 -lmsimg32 -lopengl32 -lwinmm -lws2_32 -lole32 -lpsapi -lmpr -lluajit -lstdc++" T="vxagent.exe" "${DIR}"/build.sh
rm $SYSO
