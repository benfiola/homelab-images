---
title: azerothcore
---

# azerothcore

A Docker image that runs [AzerothCore](https://www.azerothcore.org/). It ships a single `azerothcore` CLI binary that acts as the container entrypoint and dispatches to one of five subcommands: `init` handles first-run bootstrapping, `authserver`, `worldserver`, and `dbimport` configure their runtime environment and then exec into the corresponding binary, and `web` serves a small admin UI for account management.

## `init`

Downloads and caches game data, waits for the database to become reachable, runs database migrations, updates the realmlist, and bootstraps an admin account and a hidden web-service account if they don't already exist. Designed to run to completion as an init container before `authserver` and `worldserver` start.

| Variable                      | Flag                  | Default | Description                                                                                       |
| ------------------------------ | ---------------------- | --------- | ----------------------------------------------------------------------------------------------------- |
| `AC_GAME_DATA_URL`            | `--game-data-url`     | —       | URL of the game data archive to download and extract. **Required.**                               |
| `AC_REALMLIST_ADDRESS`        | `--realmlist-address` | —       | Address written to the realmlist (e.g. a DNS name or IP). If unset the realmlist is not modified. |
| `AC_ADMIN_USERNAME`           | `--admin-username`    | `admin` | Username of the bootstrapped admin account (see below).                                           |
| `AC_DATA_DIR`                 | —                      | `/data` | Directory where game data is stored.                                                               |
| `AC_LOGIN_DATABASE_INFO`      | —                      | —       | Auth database connection string (`host;port;user;pass;dbname`). **Required.**                      |
| `AC_WORLD_DATABASE_INFO`      | —                      | —       | World database connection string (`host;port;user;pass;dbname`). **Required.**                     |
| `AC_CHARACTER_DATABASE_INFO`  | —                      | —       | Character database connection string (`host;port;user;pass;dbname`). **Required.**                 |
| `AC_PLAYERBOTS_DATABASE_INFO` | —                      | —       | Playerbots database connection string (`host;port;user;pass;dbname`). **Required.**                |

### Bootstrap accounts

On first run, `init` creates two GM level 3 ("Administrator") accounts directly against the auth database if they don't already exist, then never touches them again — from that point on, all account management (including for these two accounts) happens through the `web` UI:

- **`AC_ADMIN_USERNAME`** (default `admin`) — the human-facing account used to log into the web UI.
- **`WEBUI`** — a hidden service account used only by the `web` command to authenticate to worldserver's SOAP interface. It's excluded from the web UI's account listing and isn't meant to be logged into directly.

Both accounts' randomly generated initial passwords are recorded in a small table, `web_bootstrap_credentials`, created in the auth database (outside of AzerothCore's own migration system). To recover an account's initial password:

```sql
SELECT * FROM web_bootstrap_credentials;
```

This table is written once per account and never updated again — if an account's password is later changed through the web UI, its row here will still show the *original* bootstrap password, not the current one.

## `authserver`

Copies the reference `authserver.conf.dist` to `authserver.conf` if no config file is present, then replaces the process with the AzerothCore `authserver` binary.

| Variable                 | Default            | Description                                                                   |
| ------------------------- | -------------------- | -------------------------------------------------------------------------------- |
| `AC_LOGIN_DATABASE_INFO` | —                  | Auth database connection string (`host;port;user;pass;dbname`). **Required.** |
| `AC_DATA_DIR`            | `/data`            | Path to the game data directory.                                              |
| `AC_LOGS_DIR`            | `/logs`            | Directory where log files are written.                                        |
| `AC_TEMP_DIR`            | `/tmp/azerothcore` | Directory used for temporary files.                                           |

## `worldserver`

Copies the reference `worldserver.conf.dist` and `playerbots.conf.dist` to their `.conf` counterparts if not already present, then replaces the process with the AzerothCore `worldserver` binary. Also enables and exposes its SOAP interface (used by the `web` command to run account management commands), overriding the `.conf.dist` defaults so it's reachable from another container without hand-editing `worldserver.conf`.

| Variable                      | Default            | Description                                                                         |
| ------------------------------ | -------------------- | ----------------------------------------------------------------------------------------- |
| `AC_LOGIN_DATABASE_INFO`      | —                  | Auth database connection string (`host;port;user;pass;dbname`). **Required.**       |
| `AC_WORLD_DATABASE_INFO`      | —                  | World database connection string (`host;port;user;pass;dbname`). **Required.**      |
| `AC_CHARACTER_DATABASE_INFO`  | —                  | Character database connection string (`host;port;user;pass;dbname`). **Required.**  |
| `AC_PLAYERBOTS_DATABASE_INFO` | —                  | Playerbots database connection string (`host;port;user;pass;dbname`). **Required.** |
| `AC_DATA_DIR`                 | `/data`            | Path to the game data directory.                                                    |
| `AC_LOGS_DIR`                 | `/logs`            | Directory where log files are written.                                              |
| `AC_TEMP_DIR`                 | `/tmp/azerothcore` | Directory used for temporary files.                                                 |
| `AC_SOAP_ENABLED`             | `1`                | Whether worldserver's SOAP interface is enabled. Forced on unless overridden.        |
| `AC_SOAP_IP`                  | `0.0.0.0`          | Address SOAP binds to. Forced to all interfaces unless overridden.                  |
| `AC_SOAP_PORT`                | `7878`              | Port SOAP listens on (AzerothCore's own default, left unmodified). `web`'s `AC_WEB_SOAP_ADDRESS` needs to target this. |

## `dbimport`

Replaces the process with the AzerothCore `dbimport` binary, which applies SQL migrations to all four databases. Because the mod-playerbots module is compiled into the image, `dbimport` will also apply playerbots-specific migrations.

| Variable                      | Default               | Description                                                                         |
| ------------------------------ | ------------------------ | ----------------------------------------------------------------------------------------- |
| `AC_LOGIN_DATABASE_INFO`      | —                     | Auth database connection string (`host;port;user;pass;dbname`). **Required.**       |
| `AC_WORLD_DATABASE_INFO`      | —                     | World database connection string (`host;port;user;pass;dbname`). **Required.**      |
| `AC_CHARACTER_DATABASE_INFO`  | —                     | Character database connection string (`host;port;user;pass;dbname`). **Required.**  |
| `AC_PLAYERBOTS_DATABASE_INFO` | —                     | Playerbots database connection string (`host;port;user;pass;dbname`). **Required.** |
| `AC_DATA_DIR`                 | `/data`               | Path to the game data directory.                                                    |
| `AC_LOGS_DIR`                 | `/logs`               | Directory where log files are written.                                              |
| `AC_TEMP_DIR`                 | `/tmp/azerothcore`    | Directory used for temporary files.                                                 |
| `AC_MY_SQLEXECUTABLE`         | resolved from `PATH` | Path to the `mysql` CLI.                                                             |

## `web`

Serves a small server-rendered admin UI for managing AzerothCore accounts. Any authenticated account can change its own password; accounts at or above the configured GM level threshold can additionally list every account (including its id, i.e. "GUID", and GM level), reset any account's password, and delete accounts. Nobody can ever view a stored password — AzerothCore only ever stores a non-reversible SRP-6 salt/verifier pair.

Account listings and password verification (login, and old-password checks for self-service changes) are read directly from the auth database. All mutations — password resets and account deletion — are performed via worldserver's SOAP interface, running the same `.account set password` / `.account delete` GM commands a human GM would type, rather than reimplementing AzerothCore's own account mutation logic (in particular, `.account delete` cascades to the account's characters and related rows).

| Variable                          | Flag                          | Default | Description                                                                                    |
| ----------------------------------- | ------------------------------- | --------- | ---------------------------------------------------------------------------------------------------- |
| `AC_WEB_LISTEN_ADDRESS`           | `--listen-address`            | `:8080` | Address the web server listens on.                                                            |
| `AC_WEB_SOAP_ADDRESS`             | `--soap-address`              | —       | Address of worldserver's SOAP interface, e.g. `worldserver:7878`. **Required.**                |
| `AC_WEB_ADMIN_GM_LEVEL_THRESHOLD` | `--admin-gm-level-threshold`  | `3`     | Minimum GM level (from `account_access`) that grants access to the admin dashboard.            |
| `AC_LOGIN_DATABASE_INFO`          | —                              | —       | Auth database connection string (`host;port;user;pass;dbname`). **Required.**                  |

`web` looks up its own SOAP credentials (the `WEBUI` service account bootstrapped by `init`) from the `web_bootstrap_credentials` table at startup, and fails fast if that row isn't present yet — `init` must have run against the same database first.
