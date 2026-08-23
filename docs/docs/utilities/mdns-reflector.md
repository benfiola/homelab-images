---
title: mdns-reflector
---

# mdns-reflector

A Docker image that runs [Avahi](https://avahi.org/) configured as a bidirectional mDNS reflector, enabling service discovery across network segments that would otherwise be isolated. On startup, the container scans `/sys/class/net` and automatically selects interfaces to participate in reflection — any interface backed by real hardware (indicated by the presence of `/sys/class/net/<iface>/device`) or literally named `mdns0` that is currently up. The detected interface list is logged at startup and written into Avahi's configuration before the daemon is started; no manual configuration is required.
