#!/bin/bash
set -e

COMMIT=309da733af3bcb233c33d51fbddcdc5f05aa99bf

if [ -z "$AC_ROOT" ]; then
    echo "error: AC_ROOT is required" >&2
    exit 1
fi

apt update
apt install -y git
rm -rf /var/lib/apt/lists/*

git clone https://github.com/claudevandort/mod-mount-scaling.git "$AC_ROOT/modules/mod-mount-scaling"
git -C "$AC_ROOT/modules/mod-mount-scaling" checkout "$COMMIT"
