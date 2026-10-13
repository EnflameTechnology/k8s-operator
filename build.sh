#!/bin/bash
#
# Copyright 2022 Enflame. All Rights Reserved.
#

if [ -f "../../../g_version.txt" ]; then
    source ../../../g_version.txt
fi
PKG_VER=${GCU_OPERATOR_2_0_VERSION:-"0.0.0"}

function usage() {
  cat <<EOF
Usage: ./build.sh [command]
Command:
    all       Build the gcu-operator package.
    clean     Clean the gcu-operator package.
Example:
    ./build.sh all
    ./build.sh clean
EOF
}

function build_all() {
    if [ -d "./dist" ]; then
        rm -rf ./dist
    fi
    echo -e "\033[33mBuilding gcu-operator package start...\033[0m"

    PKG_DIR="dist/gcu-operator_${PKG_VER}"
    mkdir -p $PKG_DIR
    go build -ldflags="-X main.Version=${PKG_VER}" -o manager main.go
    mv manager ${PKG_DIR}

    cp -rf -L deployments/* ${PKG_DIR}
    cp -rf enflame-resources ${PKG_DIR}
    cd ${PKG_DIR}
    build_binarys
    cd -

    echo -e "\033[33mBuilding gcu-operator package successfully!\033[0m"
}

function build_binarys() {
    mkdir -p component-images/driver/binary
    go build -o component-images/driver/binary/driver-toolkit ../../components/driver/main.go

    mkdir -p component-images/toolkit/binary
    go build -o component-images/toolkit/binary/enflame-toolkit ../../components/container-toolkit/main.go

    mkdir -p component-images/plugin/binary
    go build -o component-images/plugin/binary/cdi-manager ../../components/cdi-manager/main.go

    mkdir tools
    go build -o tools/jy-read ../../pkg/tools/jy_read.go
}

function clean_all() {
    if [ -d "./dist" ]; then
        rm -rf ./dist
    fi
    echo -e "\033[33mClean gcu-operator package successfully.\033[0m"
}

function main() {

    [[ "$EUID" -ne 0 ]] && { echo "[INFO] Run as non-root"; }
    [[ "$#" -lt 1 ]] && { usage >&2; exit 1; }

    case $1 in
    "clean")
        clean_all;;
    "all")
        build_all;;
    *)
        usage
        exit 1;;
    esac
}

main "$@"
