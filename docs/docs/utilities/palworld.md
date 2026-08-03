---
title: palworld
---

# palworld

A Docker image that downloads and runs a [Palworld](https://tech.palworldgame.com/) dedicated server (fetched via Steam on first boot and cached), with a small admin web UI for editing gameplay/balance settings and rebooting the server to apply them - without restarting the container. "Server management" settings (`SERVER_NAME`, `ADMIN_PASSWORD`, etc.) are set via env var/flag and not editable through the UI, by design; everything else (world/gameplay balance) is edited live through the UI. See the [Palworld Server Guide](https://tech.palworldgame.com/optimize-game-balance) for the full settings reference and REST API docs.

## Environment Variables

| Variable                | Default                   | Description                                                                                                               |
| ----------------------- | ------------------------- | ------------------------------------------------------------------------------------------------------------------------- |
| `CACHE_PATH`            | `/cache`                  | Directory where the downloaded game is cached (squashfs, content-addressed by Steam manifest ID)                          |
| `DATA_PATH`             | `/data`                   | Persistent server data directory (saves, config, backups)                                                                 |
| `GAME_PATH`             | `/game`                   | Directory where the game is installed - ephemeral, re-derived from the cache on every container start                     |
| `MANIFEST_ID`           | —                         | Steam manifest ID to pin a specific game version                                                                          |
| `PORT`                  | `8211`                    | Game server UDP port                                                                                                      |
| `ADMIN_PASSWORD`        | `""`                      | Also the web UI's login password (same credential used for the in-game admin/REST API). UI is disabled until this is set. |
| `SERVER_NAME`           | `Default Palworld Server` | Server name shown to connecting players                                                                                   |
| `SERVER_PASSWORD`       | `""`                      | Password required to join the server                                                                                      |
| `PLAYERS`               | `32`                      | Max concurrent players                                                                                                    |
| `TZ`                    | `UTC`                     | Timezone used for history snapshot timestamps in the UI                                                                   |
| `ADMIN_ADDRESS`         | `:8080`                   | Address the admin web UI listens on                                                                                       |
| `ADMIN_HISTORY_LIMIT`   | `20`                      | Number of past config snapshots kept for restore                                                                          |
| `UPDATE_CHECK_INTERVAL` | `0` (disabled)            | How often to check for a new game version and auto-reboot to apply it (e.g. `1h`)                                         |
| `PAUSE_ENABLED`         | `true`                    | Freeze the server when no players are connected, to save CPU/RAM                                                          |
| `PAUSE_IDLE_TIMEOUT`    | `5m`                      | How long the server must be empty before pausing                                                                          |
