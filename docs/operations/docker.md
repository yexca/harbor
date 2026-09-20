# Docker Deployment

Requires Docker Engine and the Compose plugin. Run from the project root:

```sh
docker compose up -d --build
docker compose ps
```

Open `http://localhost:7750` on the host, or substitute its reachable hostname/IP
on another device. Set the port and optional editing password through
[configuration](configuration.md).

The root [docker-compose.yml](../../docker-compose.yml) builds `harbor:local`
from source; this name is not a published
registry image. Docker performs the Go build, so the host needs neither Go nor
Node.js. The production image contains one executable and CA certificates on
Alpine, and runs as UID/GID `10001:10001`.

Compose applies a read-only root filesystem, drops all capabilities, and enables
`no-new-privileges`. `/data` is the writable named volume. Compose prefixes its
actual volume name with the project name; use `docker volume ls` to inspect
the installation.

## Plain Docker

```sh
docker build -t harbor:local .
docker run -d --name harbor \
  -p 7750:7750 \
  -v harbor-data:/data \
  -e HARBOR_ADMIN_PASSWORD=replace-with-your-password \
  --read-only --cap-drop ALL --security-opt no-new-privileges:true \
  --restart unless-stopped harbor:local
```

The multiline examples use a POSIX shell. In PowerShell, put the command on one
line or use PowerShell line continuation.

## NAS Storage and Updates

For a NAS directory, replace `harbor-data:/data` with `./data:/data` and make it
writable by UID/GID `10001:10001`. Start only one instance against that directory.
Back up `services.json` before upgrading. Then rebuild/recreate the service:

```sh
docker compose up -d --build
docker compose logs --tail=100
```

`docker compose down` preserves the named volume. Adding `-v` deletes the volume
and saved collection. See [backup and restore](data.md).

The image's health check runs `/harbor healthcheck`, which requests `/healthz`.
This verifies the Harbor process, not the availability of configured services.

## Architectures

The Dockerfile supports Linux AMD64 and ARM64 builds. Build and load one target
locally, for example:

```sh
docker buildx build --platform linux/arm64 -t harbor:arm64 --load .
```

Version tags can publish multi-platform images to GHCR and Docker Hub after CI
passes. Configure the Docker Hub token using the
[release guide](../development/commit-and-release.md). The default Compose file
continues to build locally; use a registry image only after its release run succeeds.
