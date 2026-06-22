#!/bin/sh
set -e

# NOTE: hack for https://github.com/devcontainers/features/issues/1662
sudo modprobe iptable_nat || true