#!/bin/sh
set -e

interfaces=""
for path in /sys/class/net/*/; do
    name=$(basename "$path")
    [ "$name" = "lo" ] && continue
    if [ -e "/sys/class/net/$name/device" ] || [ "$name" = "mdns0" ]; then
        operstate=$(cat "/sys/class/net/$name/operstate" 2>/dev/null || echo "unknown")
        [ "$operstate" = "up" ] && interfaces="${interfaces:+$interfaces,}$name"
    fi
done

echo "Detected interfaces for mDNS reflection: ${interfaces:-none}"

if [ -z "$interfaces" ]; then
    echo "No suitable interfaces detected" >&2
    exit 1
fi

cat > /etc/avahi/avahi-daemon.conf <<EOF
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

exec avahi-daemon --no-drop-root --no-chroot
