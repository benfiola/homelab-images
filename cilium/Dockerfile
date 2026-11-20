# Upstream cilium-agent image with a patch applied, built the same way as
# images/cilium/Dockerfile upstream (amd64 only). The three image refs below are
# copied from that file at CILIUM_VERSION and must be re-synced when it changes.
# renovate: datasource=github-releases depName=cilium/cilium
ARG CILIUM_VERSION=v1.20.5
ARG CILIUM_BUILDER_IMAGE=quay.io/cilium/cilium-builder:4c4f47fa3111779e4edf07f6c2a182925b1a8d82@sha256:aa0c951a0ff66cb7eedadf0f75e61037255478f0f31b54b1dc94c2b0aec9919c
ARG CILIUM_RUNTIME_IMAGE=quay.io/cilium/cilium-runtime:9a557f2c4cbbd9b7e2a693b81a7922590f2ce93a@sha256:565b95fce19eadf50d35095aad234be3945fe8ca1ded5fe03cdd817b51107fea
ARG CILIUM_ENVOY_IMAGE=quay.io/cilium/cilium-envoy:v1.39.6-1794557904-dee711ff0e84b106edd0abcdd71961af1b870359@sha256:bdef13043ebaa7b9dcd8e3a86be41080bb5369e017cd759a12daf89e2b8fafe7

FROM ${CILIUM_ENVOY_IMAGE} AS cilium-envoy

FROM ${CILIUM_BUILDER_IMAGE} AS builder

ARG CILIUM_VERSION

RUN git clone --depth 1 --branch "${CILIUM_VERSION}" https://github.com/cilium/cilium.git /tmp/cilium && \
    cp -a /tmp/cilium/. /go/src/github.com/cilium/cilium/ && \
    rm -rf /tmp/cilium

WORKDIR /go/src/github.com/cilium/cilium

COPY patches /tmp/patches
RUN for p in /tmp/patches/*.patch; do git apply -v "$p"; done

RUN --mount=type=cache,target=/root/.cache \
    --mount=type=cache,target=/go/pkg \
    make DESTDIR=/tmp/install PKG_BUILD=1 build-container install-container-binary install-bash-completion licenses-all && \
    mv LICENSE.all /tmp/install/LICENSE.all && \
    cp images/cilium/init-container.sh \
       plugins/cilium-cni/install-plugin.sh \
       plugins/cilium-cni/cni-uninstall.sh \
       /tmp/install/

FROM ${CILIUM_RUNTIME_IMAGE}

RUN echo ". /etc/profile.d/bash_completion.sh" >> /etc/bash.bashrc
COPY --from=cilium-envoy /usr/bin/cilium-envoy /usr/bin/cilium-envoy-starter /usr/bin/
ENV HUBBLE_SERVER=unix:///var/run/cilium/hubble.sock
COPY --from=builder /tmp/install /
WORKDIR /home/cilium

ENV INITSYSTEM="SYSTEMD"
CMD ["/usr/bin/cilium-dbg"]
