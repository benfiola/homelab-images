#!/bin/bash
set -e

COMMIT=2ddf6ff75bdbfee3c81f2c149a07126f1d0bf200

if [ -z "$AC_ROOT" ]; then
    echo "error: AC_ROOT is required" >&2
    exit 1
fi

apt update
apt install -y git
rm -rf /var/lib/apt/lists/*

git clone https://github.com/azerothcore/mod-aoe-loot.git "$AC_ROOT/modules/mod-aoe-loot"
git -C "$AC_ROOT/modules/mod-aoe-loot" checkout "$COMMIT"
