#!/bin/bash
set -e

COMMIT=aa44c28acb21fdc1d45084788214f2eee2d5f87c

if [ -z "$AC_ROOT" ]; then
    echo "error: AC_ROOT is required" >&2
    exit 1
fi

apt update
apt install -y git
rm -rf /var/lib/apt/lists/*

git clone https://github.com/azerothcore/mod-npc-services.git "$AC_ROOT/modules/mod-npc-services"
git -C "$AC_ROOT/modules/mod-npc-services" checkout "$COMMIT"
