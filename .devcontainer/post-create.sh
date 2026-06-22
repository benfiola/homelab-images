#!/bin/sh
set -e

apt -y update
DEBIAN_FRONTEND=noninteractive apt -y install make

