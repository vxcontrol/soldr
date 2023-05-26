#/bin/bash

set -e

if [ "$#" -ne 1 ]; then
    docker run -it --rm -v $(pwd):/tmp/deps -w /tmp/deps debian:buster bash -c "/tmp/deps/build_deps.sh libraries_amd64.tar.gz"
    docker run -it --rm -v $(pwd):/tmp/deps -w /tmp/deps i386/debian:buster bash -c "/tmp/deps/build_deps.sh libraries_386.tar.gz"
    echo ">>> dependency libraries was updated successful"
    ls -lah libraries_*
    md5sum libraries_*
    echo
    exit 0
fi

ARCHIVE_NAME=$1
ARCHIVE_PATH="$(pwd)/$1"
LIBRARIES_PATH=/usr/lib/vxagent/
LIBRARIES_NAMES=(
    'ld-linux'
    'libcrypto.so.1.1'
    'libc.so.6'
    'libdl.so.2'
    'libgcc_s.so.1'
    'libm.so.6'
    'libnsl.so.1'
    'libnss_compat.so.2'
    'libnss_dns.so.2'
    'libnss_files.so.2'
    'libnss_hesiod.so.2'
    'libnss_nisplus.so.2'
    'libnss_nis.so.2'
    'libnss_sss.so.2'
    'libpthread.so.0'
    'libresolv.so.2'
    'librt.so.1'
    'libssl.so.1.1'
    'libstdc++.so.6'
)
LIBRARIES_SYM_LINKS=(
    'ld-linux:ld-2.28.so'
    'libc.so.6:libc-2.28.so'
    'libc.so.6:libc.so'
    'libdl.so.2:libdl-2.28.so'
    'libdl.so.2:libdl.so'
    'libm.so.6:libm-2.28.so'
    'libm.so.6:libm.so'
    'libnsl.so.1:libnsl-2.28.so'
    'libnsl.so.1:libnsl.so'
    'libnss_compat.so.2:libnss_compat.so'
    'libnss_dns.so.2:libnss_dns.so'
    'libnss_files.so.2:libnss_files.so'
    'libnss_hesiod.so.2:libnss_hesiod.so'
    'libnss_nisplus.so.2:libnss_nisplus.so'
    'libnss_nis.so.2:libnss_nis.so'
    'libnss_sss.so.2:libnss_sss.so'
    'libpthread.so.0:libpthread.so'
    'libresolv.so.2:libresolv-2.28.so'
    'libresolv.so.2:libresolv.so'
    'librt.so.1:librt-2.28.so'
    'librt.so.1:librt.so'
)
LIBRARIES_APT_PACKAGES=(
    'libnss-sss'
    'libssl1.1'
)
echo ">>> started building"
uname -a
echo ">>> libraries archive name: $ARCHIVE_NAME"

function prepare_env {
    apt update >/dev/null 2>&1
    apt install -y --no-install-recommends ${LIBRARIES_APT_PACKAGES[@]} >/dev/null 2>&1
    mkdir -p $LIBRARIES_PATH
    echo ">>> environment was prepared"
}

function update_sym_links {
    for sym_link_data in "${LIBRARIES_SYM_LINKS[@]}"; do
        local sym_link_pair
        IFS=':' read -r -a sym_link_pair <<< "$sym_link_data"
        local sym_link_src=${sym_link_pair[0]}
        local sym_link_dst=${sym_link_pair[1]}
        if [ "$1" = "$sym_link_src" ]; then
            echo "    use symlink: $sym_link_src => $sym_link_dst"
            sym_link_src=${LIBRARIES_PATH}${2}
            sym_link_dst=${LIBRARIES_PATH}${sym_link_dst}
            ln -s $sym_link_src $sym_link_dst
            echo "      files linked: $sym_link_src => $sym_link_dst"
        fi
    done
}

function copy_library {
    local lib_path=$(ldconfig -p | grep $1 | head -n 1 | awk -F' => ' '{ print $2 }')
    echo "    library: '$1' => '$lib_path'"
    cp $lib_path $LIBRARIES_PATH
    local lib_name=$(basename ${lib_path})
    echo "    stat: $(md5sum ${LIBRARIES_PATH}${lib_name})"
    update_sym_links $1 $lib_name
}

function build_archive {
    rm -f $ARCHIVE_PATH
    pushd $LIBRARIES_PATH >/dev/null 2>&1
        chmod -R 755 .
        tar -czf $ARCHIVE_PATH .
    popd >/dev/null 2>&1
    echo "  archive size is '$(du -sh $ARCHIVE_PATH | cut -f1)'"
    echo "  archive hash is '$(md5sum $ARCHIVE_PATH | cut -d ' ' -f1)'"
}

echo ">>> starting to prepare environment"
prepare_env
echo ">>> starting to copy libraries"
for lib_name in "${LIBRARIES_NAMES[@]}"; do
    echo "  try to copy library '$lib_name'"
    copy_library $lib_name
done
echo ">>> libraries was copied"
echo ">>> starting to build archive '$ARCHIVE_NAME'"
build_archive
echo ">>> building archive '$ARCHIVE_PATH' has done"
echo
