#!/bin/bash
set -e

if [ -z "$AC_ROOT" ]; then
    echo "error: AC_ROOT is required" >&2
    exit 1
fi

apt update
apt install -y \
    build-essential \
    clang \
    cmake \
    libboost-all-dev \
    libbz2-dev \
    libmysqlclient-dev \
    libncurses-dev \
    libreadline-dev \
    libssl-dev
rm -rf /var/lib/apt/lists/*

mkdir -p "$AC_ROOT/build"
cd "$AC_ROOT/build"
cmake .. \
    -DCMAKE_INSTALL_PREFIX="$AC_ROOT/env/dist" \
    -DCMAKE_BUILD_TYPE=RelWithDebInfo \
    -DCMAKE_C_COMPILER=clang \
    -DCMAKE_CXX_COMPILER=clang++ \
    -DCMAKE_CXX_FLAGS="-w" \
    -DAPPS_BUILD=all \
    -DTOOLS_BUILD=all \
    -DSCRIPTS=static \
    -DMODULES=static
make -j$(nproc) install
