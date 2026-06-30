#!/bin/bash
set -e

COMMIT=049f35906ed0b66ea5fcdd7fdc9f12ca2ab480ca

if [ -z "$AC_ROOT" ]; then
    echo "error: AC_ROOT is required" >&2
    exit 1
fi

apt update
apt install -y git
rm -rf /var/lib/apt/lists/*

git clone https://github.com/mod-playerbots/mod-playerbots.git "$AC_ROOT/modules/mod-playerbots"
git -C "$AC_ROOT/modules/mod-playerbots" checkout "$COMMIT"
