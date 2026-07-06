#!/bin/bash
set -e

COMMIT=2134690bb03899e5c9e44d0682e8e6abf0bbbaf2

if [ -z "$AC_ROOT" ]; then
    echo "error: AC_ROOT is required" >&2
    exit 1
fi

apt update
apt install -y git
rm -rf /var/lib/apt/lists/*

git clone https://github.com/noisiver/mod-junk-to-gold.git "$AC_ROOT/modules/mod-junk-to-gold"
git -C "$AC_ROOT/modules/mod-junk-to-gold" checkout "$COMMIT"
