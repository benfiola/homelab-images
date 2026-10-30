---
title: Environment Setup
sidebar_position: 1
---

This guide provides instructions for setting up a development environment and contributing to the homelab-images project.

## Prerequisites

Before getting started, ensure you have:

- [Nix](https://nixos.org/download) with flakes enabled (provides Go, Node.js, Helm and the other tooling), or Docker if using devcontainers
- **Git** for version control
- A code editor (VSCode recommended)

## Quick Start

### Option 1: Using Devcontainers (Recommended)

The project includes a devcontainer that installs Nix and loads the flake's dev shell in every terminal.

1. **Open the project in VSCode** with the [Dev Containers extension](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-containers)
2. **Reopen in Container**: Press `Ctrl+Shift+P` (or `Cmd+Shift+P` on macOS) and select "Dev Containers: Reopen in Container"
3. **Wait for setup**: The first start downloads the Nix dependencies; they are cached in a volume afterwards

### Option 2: Local Development Setup

If you prefer local development without devcontainers:

1. **Enter the dev shell** from the repository root:

```bash
nix develop
```

2. **Install svu** (installed from a fork, so it isn't part of the flake):

```bash
make install-svu
export PATH="$PATH:$PWD/.bin"
```

## Documentation Development

The project uses [Docusaurus](https://docusaurus.io/) for documentation.

### Local Documentation Server

Start the development server:

```bash
make dev-docs
```

This starts a local server on `localhost:3000` with live reloading.

## Code Formatting

The project uses `gofmt` for formatting Go files and [Prettier](https://prettier.io/) for everything else. Format on save is configured automatically in devcontainers.
