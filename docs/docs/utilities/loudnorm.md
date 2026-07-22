---
title: loudnorm
---

# loudnorm

Loudness normalize media files in your library. This tool integrates with Sonarr and Radarr via webhooks to automatically process new or upgraded media files. It uses an SSD-based scratch directory for processing to minimize I/O to NFS-based media storage, ensuring consistent loudness normalization (EBU R128 standard -23 LUFS by default) across all content in your Jellyfin library.

## Environment Variables

| Variable | Required | Default | Description |
|---|---|---|---|
| `MEDIA_DIR` | Yes | - | Root directory containing all media files to normalize |
| `DATA_DIR` | Yes | - | Directory for state database (bbolt) and scratch storage |
| `LISTEN_ADDR` | No | `:8080` | HTTP server address for webhooks and health check |
| `WEBHOOK_TOKEN` | No | - | Optional bearer token for webhook authentication |
| `LOUDNESS_TARGET` | No | `-23` | Target loudness in LUFS (EBU R128 standard) |
| `WORKERS` | No | `1` | Number of concurrent processing workers |
| `SCAN_INTERVAL` | No | `30m` | How often to scan media directory for new files |
| `LOG_LEVEL` | No | `info` | Log level (debug, info, warn, error) |
| `LOG_FORMAT` | No | `text` | Log format (text or json) |

## Webhook Setup

### Sonarr

1. Settings → Connections → Add Webhook Notification
2. Name: `loudnorm`
3. URL: `http://<loudnorm-host>:8080/sonarr`
4. Authorization Header: `Authorization: Bearer <WEBHOOK_TOKEN>` (if token configured)
5. Events: Select:
   - On Import Complete
   - On Episode File Delete
   - On Episode File Delete for Upgrade
   - On Rename

### Radarr

1. Settings → Connections → Add Webhook Notification
2. Name: `loudnorm`
3. URL: `http://<loudnorm-host>:8080/radarr`
4. Authorization Header: `Authorization: Bearer <WEBHOOK_TOKEN>` (if token configured)
5. Events: Select:
   - On File Import
   - On File Upgrade
   - On Movie File Delete
   - On Movie File Delete for Upgrade
   - On Rename

## How It Works

- Files are copied to scratch storage (SSD) for processing
- Audio is extracted and analyzed for loudness levels
- Two-pass loudness normalization is performed using ffmpeg
- Video codec is preserved (no re-encoding); audio is normalized to AAC 256 kbps
- Subtitles and other streams are preserved
- Processed files are atomically moved back to media location
- State is tracked in bbolt database to avoid re-processing
