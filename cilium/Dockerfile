# Upstream cilium-agent image with a patch applied, built the same way as
# images/cilium/Dockerfile upstream (amd64 only). The three image refs below are
# copied from that file at CILIUM_VERSION and must be re-synced when it changes.
# renovate: datasource=github-releases depName=cilium/cilium
ARG CILIUM_VERSION=v1.20.4
ARG CILIUM_BUILDER_IMAGE=quay.io/cilium/cilium-builder:66897bc214ae487c12578033551e0404996a88e7@sha256:3e560a0977aecc72f1dca43d0e8e101fbc5415e22a638dd61ebf29f44f43920f
ARG CILIUM_RUNTIME_IMAGE=quay.io/cilium/cilium-runtime:d3ef3d6b993da541b58cb69b832f432d8d753059@sha256:577a057256325f139db1acb598d73c0cf7cd081f4bcc8ecb942c271a8ca32dbd
ARG CILIUM_ENVOY_IMAGE=quay.io/cilium/cilium-envoy:v1.39.5-1792733054-8c73b2816bc8239e27f1fd53b271884688137459@sha256:547ab45f6092f681d9f54f3be9fa8fb1e705c9c5dc2df388efe3a52aa6965629

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
