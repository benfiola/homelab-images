#!/bin/bash
set -e

COMMIT=532a6ba80c31b673338fcdb747cb2226bec4887e

if [ -z "$AC_ROOT" ]; then
    echo "error: AC_ROOT is required" >&2
    exit 1
fi

apt update
apt install -y git
rm -rf /var/lib/apt/lists/*

git clone https://github.com/azerothcore/mod-npc-buffer.git "$AC_ROOT/modules/mod-npc-buffer"
git -C "$AC_ROOT/modules/mod-npc-buffer" checkout "$COMMIT"
