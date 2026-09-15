# Troubleshooting

| Symptom | Check |
| --- | --- |
| Harbor will not start | Read `docker compose logs --tail=100`; check JSON validity, permissions, and port conflicts. |
| Saved services disappear after recreation | Confirm the same Compose project and volume/bind mount are being used. Different project names create different named volumes. |
| A card is absent from home | Open All apps and enable Show on home. Hidden cards remain searchable there. |
| Changes cannot be saved | Unlock editing, check free space, the 500-service limit, and UID 10001 write access to `/data`. |
| Auto-fetch fails | Check NAS reachability, login requirements, certificate trust, icon format/size, and the destination restrictions. Upload a PNG as a fallback. |
| `localhost` service icons fail in Docker | Use a reachable NAS address or Docker service hostname; loopback icon destinations are blocked. |
| Icon fetching works but the card will not open | The browser and NAS have different network access. Save the address reachable from the browsing device. |
| Editing access disappears | Sessions expire after 24 hours and on process restart. Unlock editing again. |
| Appearance differs on another device | Theme, background, and clock format are stored per browser. Home selection is stored on the NAS. |
| Asset edits do not appear | Rebuild/restart Go or the Docker image; assets are embedded at compilation. |
| A reverse-proxy page or mutation fails | Use a dedicated host root, preserve Host, and avoid a path-prefix rewrite. |

`GET /healthz` checks process liveness only. For safe recovery, follow
[data and reliability](data.md). Redact addresses, paths, cookies, and credentials
before sharing logs or reproduction details.
