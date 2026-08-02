#!/bin/bash
set -e

COMMIT=372814b04f9316ff9aefbbd0dc642b7921bc6fe8

if [ -z "$AC_ROOT" ]; then
    echo "error: AC_ROOT is required" >&2
    exit 1
fi

apt update
apt install -y git
rm -rf /var/lib/apt/lists/*

git clone https://github.com/DustinHendrickson/mod-player-bot-reset.git "$AC_ROOT/modules/mod-player-bot-reset"
git -C "$AC_ROOT/modules/mod-player-bot-reset" checkout "$COMMIT"
