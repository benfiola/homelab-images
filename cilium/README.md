# cilium

**Temporary.** Upstream cilium-agent `v1.20.5` with [cilium/cilium#48031](https://github.com/cilium/cilium/pull/48031) applied. Remove once cilium 1.21.0 lands.

Built like upstream's `images/cilium/Dockerfile` (amd64 only). When bumping `CILIUM_VERSION`, re-sync the image refs from upstream and check the patch still applies. Renovate ignores this Dockerfile.
