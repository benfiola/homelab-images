FROM debian:bookworm-slim

ARG TARGETARCH
ARG TARGETPLATFORM

COPY $TARGETPLATFORM/bucket-sync /usr/bin/bucket-sync

ENTRYPOINT ["bucket-sync"]
