#!/bin/bash
set -e

COMMIT=5b87ed09d54548ce4b98b8d6fce879c9cf4f7122

if [ -z "$AC_ROOT" ]; then
    echo "error: AC_ROOT is required" >&2
    exit 1
fi

apt update
apt install -y git
rm -rf /var/lib/apt/lists/*

git clone https://github.com/ZhengPeiRu21/mod-individual-progression.git "$AC_ROOT/modules/mod-individual-progression"
git -C "$AC_ROOT/modules/mod-individual-progression" checkout "$COMMIT"
