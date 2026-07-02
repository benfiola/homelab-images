---
title: azerothcore
---

# azerothcore

A Docker image that runs [AzerothCore](https://www.azerothcore.org/). It ships a single `azerothcore` CLI binary that acts as the container entrypoint and dispatches to one of four subcommands: `init` handles first-run bootstrapping, meanwhile `authserver`, `worldserver`, and `dbimport` configure their runtime environment and then exec into the corresponding binary.

## `init`

Downloads and caches game data, waits for the database to become reachable, runs database migrations, updates the realmlist, and synchronises accounts from the config file. Designed to run to completion as an init container before `authserver` and `worldserver` start.

| Variable                      | Flag                  | Default               | Description                                                                                        |
| ------------------------------ | ---------------------- | ----------------------- | ----------------------------------------------------------------------------------------------------- |
| `AC_GAME_DATA_URL`            | `--game-data-url`     | —                      | URL of the game data archive to download and extract. **Required.**                                |
| `AC_REALMLIST_ADDRESS`        | `--realmlist-address` | —                      | Address written to the realmlist (e.g. a DNS name or IP). If unset the realmlist is not modified.  |
| `AC_CONFIG`                   | `--config`            | `/config/config.yaml` | Path to the account config file.                                                                    |
| `AC_DATA_DIR`                 | —                      | `/data`                | Directory where game data is stored.                                                                |
| `AC_LOGIN_DATABASE_INFO`      | —                      | —                      | Auth database connection string (`host;port;user;pass;dbname`). **Required.**                       |
| `AC_WORLD_DATABASE_INFO`      | —                      | —                      | World database connection string (`host;port;user;pass;dbname`). **Required.**                      |
| `AC_CHARACTER_DATABASE_INFO`  | —                      | —                      | Character database connection string (`host;port;user;pass;dbname`). **Required.**                  |
| `AC_PLAYERBOTS_DATABASE_INFO` | —                      | —                      | Playerbots database connection string (`host;port;user;pass;dbname`). **Required.**                 |

### Account config

The config file declares the desired set of accounts. On each `init` run, accounts present in the file are upserted and any accounts in the database that are not in the file are deleted.

```yaml
accounts:
  - username: admin
    password: secret
    gm_level: 3
```

## `authserver`

Copies the reference `authserver.conf.dist` to `authserver.conf` if no config file is present, then replaces the process with the AzerothCore `authserver` binary.

| Variable                 | Default            | Description                                                                   |
| ------------------------- | -------------------- | -------------------------------------------------------------------------------- |
| `AC_LOGIN_DATABASE_INFO` | —                  | Auth database connection string (`host;port;user;pass;dbname`). **Required.** |
| `AC_DATA_DIR`            | `/data`            | Path to the game data directory.                                              |
| `AC_LOGS_DIR`            | `/logs`            | Directory where log files are written.                                        |
| `AC_TEMP_DIR`            | `/tmp/azerothcore` | Directory used for temporary files.                                           |

## `worldserver`

Copies the reference `worldserver.conf.dist` and `playerbots.conf.dist` to their `.conf` counterparts if not already present, then replaces the process with the AzerothCore `worldserver` binary.

| Variable                      | Default            | Description                                                                         |
| ------------------------------ | -------------------- | --------------------------------------------------------------------------------------- |
| `AC_LOGIN_DATABASE_INFO`      | —                  | Auth database connection string (`host;port;user;pass;dbname`). **Required.**       |
| `AC_WORLD_DATABASE_INFO`      | —                  | World database connection string (`host;port;user;pass;dbname`). **Required.**      |
| `AC_CHARACTER_DATABASE_INFO`  | —                  | Character database connection string (`host;port;user;pass;dbname`). **Required.**  |
| `AC_PLAYERBOTS_DATABASE_INFO` | —                  | Playerbots database connection string (`host;port;user;pass;dbname`). **Required.** |
| `AC_DATA_DIR`                 | `/data`            | Path to the game data directory.                                                    |
| `AC_LOGS_DIR`                 | `/logs`            | Directory where log files are written.                                              |
| `AC_TEMP_DIR`                 | `/tmp/azerothcore` | Directory used for temporary files.                                                 |

## `dbimport`

Replaces the process with the AzerothCore `dbimport` binary, which applies SQL migrations to all four databases. Because the mod-playerbots module is compiled into the image, `dbimport` will also apply playerbots-specific migrations.

| Variable                      | Default               | Description                                                                         |
| ------------------------------ | ------------------------ | --------------------------------------------------------------------------------------- |
| `AC_LOGIN_DATABASE_INFO`      | —                     | Auth database connection string (`host;port;user;pass;dbname`). **Required.**       |
| `AC_WORLD_DATABASE_INFO`      | —                     | World database connection string (`host;port;user;pass;dbname`). **Required.**      |
| `AC_CHARACTER_DATABASE_INFO`  | —                     | Character database connection string (`host;port;user;pass;dbname`). **Required.**  |
| `AC_PLAYERBOTS_DATABASE_INFO` | —                     | Playerbots database connection string (`host;port;user;pass;dbname`). **Required.** |
| `AC_DATA_DIR`                 | `/data`               | Path to the game data directory.                                                    |
| `AC_LOGS_DIR`                 | `/logs`               | Directory where log files are written.                                              |
| `AC_TEMP_DIR`                 | `/tmp/azerothcore`    | Directory used for temporary files.                                                 |
| `AC_MY_SQLEXECUTABLE`         | resolved from `PATH` | Path to the `mysql` CLI.                                                             |
