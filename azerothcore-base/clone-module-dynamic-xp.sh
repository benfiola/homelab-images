#!/bin/bash
set -e

COMMIT=06d5d370aa7d8d8644ed241f03a986e28195c2f4

if [ -z "$AC_ROOT" ]; then
    echo "error: AC_ROOT is required" >&2
    exit 1
fi

apt update
apt install -y git
rm -rf /var/lib/apt/lists/*

git clone https://github.com/azerothcore/mod-dynamic-xp.git "$AC_ROOT/modules/mod-dynamic-xp"
git -C "$AC_ROOT/modules/mod-dynamic-xp" checkout "$COMMIT"
