---
title: single-player-tarkov
---

# single-player-tarkov

A Docker image that manages a Single Player Tarkov (SPT) server. Game content and mods are downloaded on first run and cached so subsequent container restarts skip re-downloading. The image handles SPT server installation at a specified version, optional mod installation from URLs, JSON config patching, and preservation of selected data subdirectories across reinstalls.

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `CACHE_PATH` | `/cache` | Directory where downloaded game and mod files are cached |
| `DATA_PATH` | `/data` | Persistent server data directory (profiles, configs, etc.) |
| `GAME_PATH` | `/game` | Directory where SPT is installed |
| `VERSION` | — | SPT version to install; uses the latest release if omitted |
| `MOD_URLS` | — | Space-separated URLs of mod archives to download and install |
| `CONFIG_PATCHES` | — | JSON map of config file paths to arrays of JSON Patch operations to apply |
| `DATA_SUBDIRS` | — | Space-separated subdirectory names under `DATA_PATH` to persist across reinstalls |
