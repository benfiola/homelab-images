---
title: Dependency Updates
sidebar_position: 2
---

[Renovate](https://docs.renovatebot.com) opens pull requests for dependency updates. It is configured in `renovate.json`, and its **Dependency Dashboard** issue lists pending updates and everything it detected.

## Tracking a dependency

Go modules, npm, GitHub Actions, devcontainers and Dockerfile `FROM` lines are tracked automatically. Anything else is tracked with a comment on the line above the version (`//` in Go):

```sh
# renovate: datasource=github-releases depName=helm/helm extractVersion=^v(?<version>.+)$
HELM_VERSION ?= 4.3.1
```

`versioning` and `extractVersion` are optional. For repositories without releases, track a commit with `datasource=git-refs depName=<repo url> branch=<branch>` above a 40-character sha.

## Pinning a dependency

Versions held back for this environment are marked so they stay visible in the dashboard without being updated:

```sh
# renovate-pinned: depName=cuda currentValue=12.4 reason=matches the host GPU driver
```

## Rule types

Each rule in `renovate.json` starts its `description` with its type:

- `RELEASES`: the commit scope an update gets. The publish workflow releases whichever component the scope names.
- `GROUPING`: updates that must land in one PR.
- `VERSIONS`: which versions are allowed.
- `OPT-IN`: listed as pending in the dashboard; the PR is only opened when ticked.
- `PINNED`: backs `renovate-pinned`.
- `PR NOTE`: reminder added to the PR body.
