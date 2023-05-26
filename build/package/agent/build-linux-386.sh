#!/bin/bash
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
LD_BUNDLE="-Wl,-rpath -Wl,/usr/lib/vxagent -Wl,--dynamic-linker=/usr/lib/vxagent/ld-2.28.so -lresolv -lnsl -lnss_files -lnss_dns -lcrypto -lssl"
GOOS=linux GOARCH=386 P=linux32 LF="-Wl,--wrap=fcntl64 -Wl,--wrap=fcntl -Wl,--whole-archive" LD="-Wl,--no-whole-archive -lcompat -pthread -lluajit -lm -ldl -lstdc++ $LD_BUNDLE" T="vxagent" "${DIR}"/build.sh
