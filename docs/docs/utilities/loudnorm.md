---
title: loudnorm
---

# loudnorm

Loudness-normalizes the primary audio track of media files using ffmpeg's two-pass `loudnorm` filter. Scans configured directories on startup and on a periodic interval, and can also be triggered instantly via Radarr/Sonarr webhooks (`POST /webhooks/radarr`, `POST /webhooks/sonarr`) on import, upgrade, and deletion. Radarr/Sonarr must see the same absolute file paths as `loudnorm` for webhooks to resolve correctly.

## Flags / Environment Variables

| Flag                    | Env                   | Default      | Description                                                                                                                         |
| ----------------------- | --------------------- | ------------ | ----------------------------------------------------------------------------------------------------------------------------------- |
| `--media-dirs`          | `MEDIA_DIRS`          | _(required)_ | Comma-separated list of absolute directories to scan for media files                                                                |
| `--config-dir`          | `CONFIG_DIR`          | _(required)_ | Directory holding the marker database (`state.db`) tracking what's been processed                                                   |
| `--target-i`            | `TARGET_I`            | `-16.0`      | Target integrated loudness (LUFS)                                                                                                   |
| `--target-tp`           | `TARGET_TP`           | `-1.5`       | Target true peak (dBTP)                                                                                                             |
| `--target-lra`          | `TARGET_LRA`          | `11.0`       | Target loudness range (LU)                                                                                                          |
| `--reprocess-salt`      | `REPROCESS_SALT`      | `""`         | Arbitrary value folded into the marker hash to force a full reprocess without changing targets                                      |
| `--rescan-interval`     | `RESCAN_INTERVAL`     | `45m`        | Periodic safety-net rescan interval; `0` disables it                                                                                |
| `--audio-backup-dir`    | `AUDIO_BACKUP_DIR`    | `""`         | Directory to preserve pristine original audio streams, so reprocessing re-derives from them instead of compounding lossy re-encodes |
| `--webhook-listen-addr` | `WEBHOOK_LISTEN_ADDR` | `:8080`      | Bind address for the webhook/health HTTP server                                                                                     |
| `--webhook-token`       | `WEBHOOK_TOKEN`       | `""`         | Optional shared token required on webhook requests (`?token=`)                                                                      |
| `--log-level`           | `LOG_LEVEL`           | `info`       | Log level: `debug`, `info`, `warn`, or `error`                                                                                      |
| `--log-format`          | `LOG_FORMAT`          | `text`       | Log format: `text` or `json`                                                                                                        |
