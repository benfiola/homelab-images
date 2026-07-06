#!/bin/bash
set -e

COMMIT=a1780785f386732527de7455aac960f630d29b5d

if [ -z "$AC_ROOT" ]; then
    echo "error: AC_ROOT is required" >&2
    exit 1
fi

apt update
apt install -y git
rm -rf /var/lib/apt/lists/*

git clone https://github.com/DustinHendrickson/mod-player-bot-level-brackets.git "$AC_ROOT/modules/mod-player-bot-level-brackets"
git -C "$AC_ROOT/modules/mod-player-bot-level-brackets" checkout "$COMMIT"
