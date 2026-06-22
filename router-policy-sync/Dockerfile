FROM debian:bookworm-slim

ARG TARGETARCH
ARG TARGETPLATFORM

COPY $TARGETPLATFORM/router-policy-sync /usr/bin/router-policy-sync

ENTRYPOINT ["router-policy-sync"]
