#!/usr/bin/env bash
set -euo pipefail

# take ownership of cache volume
chown -R "$(id -u):$(id -g)" /go

. /nix/var/nix/profiles/default/etc/profile.d/nix-daemon.sh

nix-shell -p git --run "git config --system --add safe.directory '*'"

# /etc/nix-devshell.sh is written by ./post-start.sh
echo '. /etc/nix-devshell.sh' | tee -a /etc/bash.bashrc >/dev/null
