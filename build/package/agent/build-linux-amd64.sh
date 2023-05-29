#!/bin/bash
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
[ "x$BUNDLE" = "xtrue" ] && LD_BUNDLE="-L/usr/lib/x86_64-linux-gnu -Wl,-rpath -Wl,/usr/lib/vxagent -Wl,--dynamic-linker=/usr/lib/vxagent/ld-2.28.so -lssp_nonshared -lc_nonshared -L/usr/lib/vxagent -lrt -lresolv -lnsl -lnss_files -lnss_dns -lnss_systemd -lcrypto -lssl"
[ "x$BUNDLE" = "xtrue" ] && T="vxbundle" || T="vxagent"
GOOS=linux GOARCH=amd64 P=linux64 LF="-Wl,--wrap=fcntl64 -Wl,--wrap=fcntl -Wl,--whole-archive" LD="-Wl,--no-whole-archive -lcompat -pthread -lluajit -lm -ldl -lstdc++ $LD_BUNDLE" T=$T "${DIR}"/build.sh
