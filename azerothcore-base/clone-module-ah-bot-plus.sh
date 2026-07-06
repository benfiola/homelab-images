#!/bin/bash
set -e

COMMIT=f685832994c825f90aa5a3dc0e1620aa568e875b

if [ -z "$AC_ROOT" ]; then
    echo "error: AC_ROOT is required" >&2
    exit 1
fi

apt update
apt install -y git
rm -rf /var/lib/apt/lists/*

git clone https://github.com/NathanHandley/mod-ah-bot-plus.git "$AC_ROOT/modules/mod-ah-bot-plus"
git -C "$AC_ROOT/modules/mod-ah-bot-plus" checkout "$COMMIT"
