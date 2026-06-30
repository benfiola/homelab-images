#!/bin/bash
set -e

COMPONENT="${1:-worldserver}"
ETC_DIR=/azerothcore/env/dist/etc
REF_DIR=/azerothcore/env/ref/etc

copy_conf_if_missing() {
    local name="$1"
    local src="${REF_DIR}/${name}.conf.dist"
    local dst="${ETC_DIR}/${name}.conf"
    if [ ! -f "$dst" ] && [ -f "$src" ]; then
        cp "$src" "$dst"
    fi
}

case "$COMPONENT" in
    init)
        exec /usr/bin/azerothcore "${@:2}"
        ;;
    authserver)
        copy_conf_if_missing authserver
        exec authserver
        ;;
    worldserver)
        copy_conf_if_missing worldserver
        copy_conf_if_missing playerbots
        exec worldserver
        ;;
    dbimport)
        exec dbimport
        ;;
    *)
        echo "Unknown component: $COMPONENT" >&2
        echo "Usage: $0 {init|authserver|worldserver|dbimport}" >&2
        exit 1
        ;;
esac
