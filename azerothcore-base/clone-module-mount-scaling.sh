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

MODULE_DIR="$AC_ROOT/modules/mod-mount-scaling"
git clone https://github.com/claudevandort/mod-mount-scaling.git "$MODULE_DIR"
git -C "$MODULE_DIR" checkout "$COMMIT"

# normalize to the "world/base" sql layout other modules use
mkdir -p "$MODULE_DIR/data/sql/world/base"
mv "$MODULE_DIR/data/sql/db_world/"*.sql "$MODULE_DIR/data/sql/world/base/"
rm -rf "$MODULE_DIR/data/sql/db_world"
