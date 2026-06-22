FROM debian:bookworm-slim

ARG TARGETARCH

COPY .build/router-policy-sync_linux_${TARGETARCH} /usr/bin/router-policy-sync

ENTRYPOINT ["router-policy-sync"]
