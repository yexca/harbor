# Local Development

## Requirements

- Go 1.26 or later, matching [go.mod](../../go.mod).
- Git and GNU Make for repository workflows.
- Node.js 24, matching [.nvmrc](../../.nvmrc), for development checks only.
- Docker with Compose for container validation and deployment.
- A compatible C compiler for `make test-race`.

There are no Go module dependencies or npm packages to install. Node.js is not
needed for `go run ./server`, the compiled application, or the production container.

## Run and Build

```sh
make run
make build
```

Run these commands from the repository root, one at a time; the server stays
in the foreground until stopped. The local URL is `http://localhost:7750` and
data defaults to `./data` relative to the working directory.
`make build` writes `bin/harbor` (`bin/harbor.exe` on Windows). Restart after web
asset changes because Go embeds assets at compilation.

For an isolated local preview in PowerShell:

```powershell
$env:HARBOR_ADDR = '127.0.0.1:18481'
$env:HARBOR_DATA_DIR = './.tmp/dev-data'
go run ./server
```

In a POSIX shell:

```sh
HARBOR_ADDR=127.0.0.1:18481 HARBOR_DATA_DIR=./.tmp/dev-data go run ./server
```

Use a dedicated terminal for preview environment variables. The server does not
load `.env`; that file is for Compose interpolation.

## Docker Development

Build and start a source checkout with the standalone development Compose file.
First create `data` in the checkout; on Linux, grant ownership to the container
user with `sudo chown 10001:10001 data`:

```sh
docker compose -f docker-compose.dev.yml -p harbor-dev up -d --build
docker compose -f docker-compose.dev.yml -p harbor-dev logs --tail=100
docker compose -f docker-compose.dev.yml -p harbor-dev down
```

It builds `harbor:local` and bind-mounts the checkout's `./data` at `/data`.
Keep production deployments in a separate directory: both Compose files use
`./data`, and different project names do not isolate that directory.
Re-run the build command after source changes;
there is no live reload. `down` preserves development data. Set `HARBOR_PORT`
in `.env` to another port if a production instance already uses port 7750.
The default `docker-compose.yml` pulls the published Docker Hub image instead.

## Repository Workflow

The root `go.mod` defines one module. `server/` contains the Go application and
tests; `server/web/` contains its embedded browser assets. Root-level `docs/`
and `scripts/` contain documentation and development tooling. Build/run targets
use `./server`; test and vet targets use `./...` to cover all module packages.

`make` lists available commands. `make ci-local` runs portable checks and a local
build; `make ci` adds race testing and an isolated Docker smoke test. See
[testing](testing.md) for narrower targets.

Recipes use direct commands without POSIX-only shell pipelines, so GNU Make can
run them on Windows. Override `GO`, `NODE`, `DOCKER`, `GO_PARALLEL`, or
`DOCKER_IMAGE` as needed. Go checks/builds default to two parallel packages.

`make docker-up` builds with `docker-compose.dev.yml` and Compose project
`harbor-dev`; the down/status/logs targets use the same configuration. Override
`COMPOSE_FILE` and `COMPOSE_PROJECT` to target another configuration or project.
`make docker-down` preserves the host data directory. The smoke target
uses its own random container name, loopback port, and anonymous volume.

See [contributing](../../CONTRIBUTING.md), [architecture](../architecture/index.md),
and [commit conventions](commit-and-release.md).
