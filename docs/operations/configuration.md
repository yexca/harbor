# Configuration

| Variable | Default | Meaning |
| --- | --- | --- |
| `HARBOR_ADDR` | `:8080` | Server listening address and port |
| `HARBOR_DATA_DIR` | `./data` locally; `/data` in Docker | Directory containing `services.json` |
| `HARBOR_ADMIN_PASSWORD` | Empty | Optional password for mutations and icon requests |
| `HARBOR_PORT` | `8080` | Host port binding in the supplied Compose file only |

The Go process reads its environment at startup. It does not load `.env` itself.
Docker Compose reads `.env` for interpolation. Restart/recreate the process after
changing configuration; changes to a Compose `.env` require `docker compose up -d`.

Copy [.env.example](../../.env.example) to `.env` and set local deployment values.
Keep the file out of Git. To restrict a local preview to the host, use
`HARBOR_PORT=127.0.0.1:8080`; use an appropriate reachable binding for NAS clients.

The provided Compose file passes the password to the container. To customize
other server variables in Compose, explicitly add them to its `environment`
mapping. Keep the container listener on port 8080 unless you also update port
mapping and health-check expectations.

Use a reverse proxy at the root of a dedicated host; a URL path prefix is not
supported. Preserve the original Host header. See [security](security.md).
