#!/bin/bash
set -e

COMMIT=62a991e783b3d2099e8f09a7dda68984d6b807f1

if [ -z "$AC_ROOT" ]; then
    echo "error: AC_ROOT is required" >&2
    exit 1
fi

apt update
apt install -y git
rm -rf /var/lib/apt/lists/*

git clone https://github.com/mod-playerbots/azerothcore-wotlk.git "$AC_ROOT"
git -C "$AC_ROOT" checkout "$COMMIT"
