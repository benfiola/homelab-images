---
title: linstor-provision-disk
---

# linstor-provision-disk

An init container for LINSTOR satellite pods that automatically detects when a satellite's ID has changed and reinitializes the storage layer accordingly — cleaning up stale LVM physical volumes, volume groups, and logical volumes before provisioning fresh storage. This makes cluster resets on satellite nodes fully hands-off.

## Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `PARTITION_LABEL` | Yes | GPT partition label identifying the storage device (e.g. `linstor-storage`) |
| `POOL` | Yes | Name of the LVM thin pool to create and manage (e.g. `linstor-pool`) |
| `SATELLITE_ID` | Yes | Stable unique identifier for this satellite node (e.g. the node name); a change triggers storage reinitialization |
| `VOLUME_GROUP` | Yes | Name of the LVM volume group to create and manage (e.g. `linstor-vg`) |
