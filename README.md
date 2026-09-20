# Harbor

A lightweight, self-hosted start page for your NAS services. A full-screen
background, clock/date, and frosted glass cards keep home quiet. Service
management stays in All apps and Settings.

Harbor uses **Go's standard library and plain HTML, CSS, and JavaScript**.
It runs as one executable with embedded local assets. There are no third-party
Go modules, frontend build tools, database servers, CDNs, or runtime Node.js
dependency.

## Features

- Responsive service cards over a local wallpaper, with a large clock and date.
- Right-click All apps panel, a final add tile, and keyboard/touch alternatives.
- Add, edit, delete, search, and choose which services appear on home.
- Custom names, addresses, descriptions, and locally stored icons.
- Website icon discovery, direct icon fetching, and image uploads.
- System/Light/Dark panels, three backgrounds, and 12/24-hour time.
- Optional password protection for editing.
- Single-file JSON persistence and Docker deployment.

The interface, source comments, and documentation are English. Each card has
one address; automatic LAN/WAN switching and service polling are outside the
current scope.

## Quick Start

Requires Docker with Compose support. Copy [docker-compose.yml](docker-compose.yml)
into an empty directory and run from there; no source checkout or build is needed:

On Linux/NAS hosts, prepare the data directory for the container user first:

```sh
mkdir -p data
sudo chown 10001:10001 data
```

```sh
docker compose up -d
```

Open `http://localhost:7750` on the host, or its reachable address on another
device. Use the top-right **Settings → Add a service**, or right-click and select
the final **+** tile in **All apps**.

To change the port or protect editing, create a `.env` file beside the Compose
file (or copy [.env.example](.env.example)), set the values, and recreate the service:

```dotenv
HARBOR_PORT=7750
HARBOR_ADMIN_PASSWORD=replace-with-your-password
```

```sh
docker compose up -d
```

Anyone who can reach Harbor can view the entire collection, including hidden
cards. Without a password, everyone can also edit it. Use a trusted network or
reverse-proxy access control and HTTPS as appropriate for your deployment.

Compose pulls `yexca/harbor:latest` from Docker Hub, with AMD64/ARM64 support.
For a fixed release, change the image tag to `yexca/harbor:v0.1.0`.
The container runs unprivileged with a read-only root filesystem and persistent
`/data`, bind-mounted from `./data` beside the Compose file. Keep backups of
`data/services.json`; `docker compose down`, including `-v`, preserves this directory.

See [Docker deployment](docs/operations/docker.md),
[configuration](docs/operations/configuration.md), and
[backup/restore](docs/operations/data.md).

## Documentation

- [Documentation map](docs/README.md) and [overview](docs/overview.md).
- [User guide](docs/user/index.md): home, All apps, editing, icons, and preferences.
- [Troubleshooting](docs/operations/troubleshooting.md).
- [Architecture](docs/architecture/index.md) and [data model](docs/architecture/data-model.md).
- [Local development](docs/development/local-dev.md) and [testing](docs/development/testing.md).
- [Design contract](DESIGN.md), [agent guide](AGENTS.md), and [contributing](CONTRIBUTING.md).
- [Privacy](PRIVACY.md), [deployment security](docs/operations/security.md), and [security reporting](SECURITY.md).
- [Asset provenance](ASSETS.md), [decisions](docs/decisions/index.md), and [history](docs/history/index.md).

## Development

To build and run from a source checkout using Docker:

```sh
docker compose -f docker-compose.dev.yml -p harbor-dev up -d --build
```

The development configuration uses `harbor:local` and the checkout's `./data`
directory. Prepare its permissions as above and rebuild after source changes.
Run production and development from separate directories so they do not share
saved data. See [local development](docs/development/local-dev.md).

Go 1.26 or later is enough to run Harbor locally:

```sh
go run ./server
```

Run this command from the repository root. Application code and embedded browser
assets live in `server/`; `go.mod` remains at the root. Assets are embedded at
compilation; restart after editing HTML, CSS, or JavaScript. Local data defaults
to `./data` relative to the working directory.

Repository checks use GNU Make, Git, and Node.js 24, with no npm packages:

```sh
make                 # List commands
make ci-local        # Portable checks and local build
make docker-smoke    # Build and test an isolated container
make sensitive-check # Review index and non-ignored working files
```

`make ci` additionally runs the Go race detector and Docker smoke checks.
The executable is written to `bin/harbor` (`bin/harbor.exe` on Windows).
See [commit and release](docs/development/commit-and-release.md) for version,
signing, and release conventions.
