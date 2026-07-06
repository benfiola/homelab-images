#!/bin/bash
set -e

COMMIT=af32add66eafc0e0eb8775999e76ceed75f18b74

if [ -z "$AC_ROOT" ]; then
    echo "error: AC_ROOT is required" >&2
    exit 1
fi

apt update
apt install -y git
rm -rf /var/lib/apt/lists/*

git clone https://github.com/azerothcore/mod-npc-enchanter.git "$AC_ROOT/modules/mod-npc-enchanter"
git -C "$AC_ROOT/modules/mod-npc-enchanter" checkout "$COMMIT"
