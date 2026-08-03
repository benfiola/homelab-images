---
title: palworld
---

# palworld

A Docker image that downloads and runs a [Palworld](https://tech.palworldgame.com/) dedicated server (fetched via Steam on first boot and cached), with a small admin web UI for editing gameplay/balance settings and rebooting the server to apply them - without restarting the container. Server management settings (`SERVER_NAME`, `ADMIN_PASSWORD`, `RCON_*`, `PUBLIC_IP`/`PUBLIC_PORT`, etc.) are env-var-driven and not editable through the UI, by design. See the [Palworld Server Guide](https://tech.palworldgame.com/) for the full settings reference and REST API docs.

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `CACHE_PATH` | `/cache` | Directory where the downloaded game is cached (squashfs, content-addressed by Steam manifest ID) |
| `DATA_PATH` | `/data` | Persistent server data directory (saves, config, backups) |
| `GAME_PATH` | `/game` | Directory where the game is installed - ephemeral, re-derived from the cache on every container start |
| `MANIFEST_ID` | — | Steam manifest ID to pin a specific game version |
| `PORT` | `8211` | Game server UDP port |
| `ADMIN_PASSWORD` | `""` | Also the web UI's login password (same credential used for the in-game admin/REST API). UI is disabled until this is set. |
| `PALWORLD_MGMT_LISTEN_ADDRESS` | `:8080` | Address the admin web UI listens on |
| `PALWORLD_MGMT_HISTORY_LIMIT` | `20` | Number of past config snapshots kept for restore |
| `UPDATE_CHECK_INTERVAL` | `0` (disabled) | How often to check for a new game version and auto-reboot to apply it (e.g. `1h`) |
