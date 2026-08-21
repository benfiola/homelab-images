---
title: music-assistant
---

# music-assistant

A Docker image extending the official [Music Assistant](https://music-assistant.io/) server with a patch that allows the stream endpoint's base URL to be overridden via an environment variable. This is useful when Music Assistant is running behind a reverse proxy or in a network where the auto-detected publish IP would produce an unreachable stream URL for players — for example, when the server is only accessible to players via a hostname rather than a bare IP address.

## Environment Variables

| Variable        | Default | Description                                                                                                                                         |
| --------------- | ------- | --------------------------------------------------------------------------------------------------------------------------------------------------- |
| `MASS_BASE_URL` | —       | Base URL advertised to players for the stream endpoint (e.g. `http://music.example.com`). When unset, Music Assistant uses its default auto-detected IP and port. |
