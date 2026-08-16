#!/bin/bash
set -e

apt-get update
apt-get install -y --no-install-recommends \
    ca-certificates \
    software-properties-common \
    wget

update-ca-certificates

add-apt-repository contrib

wget -qO /tmp/cuda-keyring.deb https://developer.download.nvidia.com/compute/cuda/repos/debian12/x86_64/cuda-keyring_1.1-1_all.deb
dpkg -i /tmp/cuda-keyring.deb
rm /tmp/cuda-keyring.deb
apt-get update
apt-get install -y --no-install-recommends \
    libcublas-12-4 \
    libcudnn9-cuda-12

mkdir -p /etc/profile.d
cat > /etc/profile.d/cuda.sh << 'CUDA_ENV'
export PATH=/usr/local/cuda-12.4/bin${PATH:+:${PATH}}
export LD_LIBRARY_PATH=/usr/local/cuda-12.4/lib64${LD_LIBRARY_PATH:+:${LD_LIBRARY_PATH}}
CUDA_ENV

test -f /usr/local/cuda-12.4/lib64/libcublas.so.12 || (echo "ERROR: libcublas.so.12 not found" && exit 1)

apt-get clean
rm -rf /var/lib/apt/lists/*