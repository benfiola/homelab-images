---
title: seven-days-to-die
---

# seven-days-to-die

A Docker image that manages a Seven Days to Die dedicated server. Game content and mods are downloaded on first run and cached so subsequent container restarts skip re-downloading. The image handles Steam-based installation, optional mod installation from URLs, deletion of default mods, and configurable auto-restart on a fixed interval.

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `CACHE_PATH` | `/cache` | Directory where downloaded game and mod files are cached |
| `DATA_PATH` | `/data` | Persistent server data directory (saves, configs, etc.) |
| `GAME_PATH` | `/game` | Directory where the game is installed |
| `MANIFEST_ID` | — | Steam manifest ID to pin a specific game version |
| `DELETE_DEFAULT_MODS` | — | If set, removes the game's default mods before installing custom ones |
| `MOD_URLS` | — | Space-separated URLs of mod archives to download and install |
| `ROOT_URLS` | — | Space-separated URLs of files to install at the game root |
| `AUTO_RESTART` | — | Duration between automatic server restarts (e.g. `24h`) |
