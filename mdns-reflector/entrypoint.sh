#!/bin/sh
set -e

interfaces=""

for path in /sys/class/net/*/; do
    name=$(basename "$path")
    [ "$name" = "lo" ] && continue
    if [ -e "/sys/class/net/$name/device" ]; then
        interfaces="${interfaces:+$interfaces,}$name"
    fi
done

if [ -n "$EXTRA_INTERFACES" ]; then
    IFS=','
    for name in $EXTRA_INTERFACES; do
        name=$(printf '%s' "$name" | tr -d ' ')
        [ -z "$name" ] && continue
        if [ -d "/sys/class/net/$name" ]; then
            interfaces="${interfaces:+$interfaces,}$name"
        fi
    done
    unset IFS
fi

echo "Detected interfaces for mDNS reflection: ${interfaces:-none}"

if [ -z "$interfaces" ]; then
    echo "No suitable interfaces detected" >&2
    exit 1
fi

cat > /tmp/avahi-daemon.conf <<EOF
[server]
allow-interfaces=$interfaces
use-ipv4=yes
use-ipv6=no
enable-dbus=no

[wide-area]
enable-wide-area=no

[publish]
disable-publishing=yes

[reflector]
enable-reflector=yes

[rlimits]
EOF

exec avahi-daemon --no-chroot -f /tmp/avahi-daemon.conf
