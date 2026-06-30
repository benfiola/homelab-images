FROM ghcr.io/benfiola/homelab-images/azerothcore-base:1.0.0

ARG TARGETARCH

COPY .build/azerothcore_linux_${TARGETARCH} /usr/bin/azerothcore

VOLUME ["/config", "/data"]

USER acore

ENTRYPOINT ["/usr/bin/azerothcore"]
