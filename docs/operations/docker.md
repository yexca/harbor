# Docker Deployment

Requires Docker Engine and the Compose plugin. Copy
[docker-compose.yml](../../docker-compose.yml) into an empty directory and run
from that directory. This file works on its own without a source checkout or `.env`:

On Linux/NAS hosts, create the host data directory and grant the container user
ownership before starting:

```sh
mkdir -p data
sudo chown 10001:10001 data
```

For Docker Desktop, create the `data` directory and allow Docker to share its
parent directory if prompted. Linux ownership commands are not needed on Windows.

```sh
docker compose up -d
docker compose ps
```

Open `http://localhost:7750` on the host, or substitute its reachable hostname/IP
on another device. Set the port and optional editing password through
[configuration](configuration.md).

The default configuration pulls `yexca/harbor:latest` from Docker Hub.
To pin a release, set `image: yexca/harbor:v0.1.0`. To use GHCR, set
`image: ghcr.io/yexca/harbor:v0.1.0`; registry authentication may be required
depending on package visibility. The host needs neither Go nor Node.js.
The production image contains one executable and CA certificates on
Alpine, and runs as UID/GID `10001:10001`.

Compose applies a read-only root filesystem, drops all capabilities, and enables
`no-new-privileges`. The host's `./data` directory beside the Compose file is
bind-mounted at `/data`; these configurations create no Docker-managed data volume.
The saved collection is `./data/services.json` on the host.

## Plain Docker

```sh
docker pull yexca/harbor:latest
docker run -d --name harbor \
  -p 7750:7750 \
  --mount type=bind,source="$(pwd)/data",target=/data \
  -e HARBOR_ADMIN_PASSWORD=replace-with-your-password \
  --read-only --cap-drop ALL --security-opt no-new-privileges:true \
  --restart unless-stopped yexca/harbor:latest
```

The multiline examples use a POSIX shell. In PowerShell, put the command on one
line or use PowerShell line continuation.

## NAS Storage and Updates

To use another NAS directory, replace `./data` in the mount with its host path
and make it writable by UID/GID `10001:10001`. Start only one instance against
that directory. Changing the Compose project name does not isolate a bind mount.
Back up `services.json` before upgrading. If pinned to a version, update the
image tag first. Then pull and recreate the service:

```sh
docker compose pull
docker compose up -d
docker compose logs --tail=100
```

`docker compose down`, including `-v`, preserves the host data directory.
See [backup, restore, and migration from a named volume](data.md).

The image's health check runs `/harbor healthcheck`, which requests `/healthz`.
This verifies the Harbor process, not the availability of configured services.

## Architectures

The Dockerfile supports Linux AMD64 and ARM64 builds. Build and load one target
locally, for example:

```sh
docker buildx build --platform linux/arm64 -t harbor:arm64 --load .
```

Version tags publish multi-platform images to GHCR and Docker Hub after CI
passes. Maintainers configure the Docker Hub token using the
[release guide](../development/commit-and-release.md).

For local source builds, use [docker-compose.dev.yml](../../docker-compose.dev.yml)
as a standalone configuration, following [local development](../development/local-dev.md).
