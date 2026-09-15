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
in the foreground until stopped. The local URL is `http://localhost:8080` and
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

`make docker-up` uses Compose project `harbor`; use `COMPOSE_PROJECT` to target
another installation. `make docker-down` preserves its volumes. The smoke target
uses its own random container name, loopback port, and anonymous volume.

See [contributing](../../CONTRIBUTING.md), [architecture](../architecture/index.md),
and [commit conventions](commit-and-release.md).
