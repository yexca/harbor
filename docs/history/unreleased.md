## Deployment

- Rename the default Compose configuration to `docker-compose.yml`.
- Pull the Docker Hub image by default for standalone production deployment.
- Add `docker-compose.dev.yml` for local builds,
  and point Makefile Compose commands at the development configuration.
- Bind-mount `./data` in both Compose configurations and document host directory
  permissions, backups, and migration from named volumes.
