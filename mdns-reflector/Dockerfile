FROM golang:1.25.8 as builder

ARG TARGETARCH
ARG TARGETPLATFORM

COPY $TARGETPLATFORM/mdns-reflector /usr/bin/mdns-reflector

ENTRYPOINT ["mdns-reflector"]
