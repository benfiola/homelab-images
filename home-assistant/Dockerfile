ARG FRIGATE_VERSION=5.15.4
ARG HA_VERSION=2025.6.3

FROM alpine AS frigate-integration

ARG FRIGATE_VERSION

COPY install-frigate-integration.sh /install.sh
RUN /install.sh "${FRIGATE_VERSION}" /custom_components

FROM ghcr.io/home-assistant/home-assistant:${HA_VERSION}

ARG TARGETARCH

COPY --from=frigate-integration /custom_components/ /custom_components/
COPY rootfs/ /
