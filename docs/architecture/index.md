# Architecture

```text
Browser -> same-origin HTTP API -> JSON store
                               -> icon fetcher -> website
```

## Modules

| Location | Responsibility |
| --- | --- |
| `server/main.go` | Server lifecycle, routes, request validation, sessions, security headers, embedded assets |
| `server/store.go` | Service identity, collection limits, site settings, background files, synchronized reads/writes, atomic replacement |
| `server/icons.go` | URL/address policy, bounded icon discovery, icon and background image validation |
| `server/web/index.html` | Page template: home screen, All apps, Settings, editors, login, and confirmation dialogs |
| `server/web/style.css` | Wallpaper, glass surfaces, themes, responsive grid, focus and motion behavior |
| `server/web/app.js` | Fetch API client, rendering, form state, dialogs, clock, browser preferences |
| `server/main_test.go` | HTTP, store, authentication, icon, and compatibility regression tests |
| `scripts/` | Development-only repository checks and isolated container smoke testing |

Application source and assets live in `server/`, with the single Go module
declared by the root `go.mod`. The Go files remain one `package main`.

The frontend has no build step. The `web/*` embed pattern in `server/main.go`
includes `server/web/` at compile time. Public asset URLs remain `/app.js`,
`/style.css`, `/favicon.svg`, and `/wallpaper.jpg`. `/` is rendered from the
`index.html` template with the saved title, icon URL, and background URL, so the
tab shows them before scripts run. A running process does not pick up edited
assets until it is rebuilt/restarted.

## Request Contracts

| Method and path | Behavior | Editing access required |
| --- | --- | --- |
| `GET /healthz` | Process liveness, plain `ok` | No |
| `GET /api/session` | `protected` and `canEdit` flags | No |
| `POST /api/login` | Accept password and create a session cookie | No |
| `POST /api/logout` | Remove the current session | No |
| `GET /api/services` | Return every service, including hidden ones | No |
| `POST /api/services` | Validate and create a service | Yes |
| `PUT /api/services/{id}` | Replace editable service fields | Yes |
| `PATCH /api/services/{id}/visibility` | Set required boolean `hidden` | Yes |
| `DELETE /api/services/{id}` | Delete a service and its stored icon | Yes |
| `POST /api/icon` | Discover or directly fetch an icon from `url`; optional `direct` | Yes |
| `GET /api/site` | Effective `title`, `icon` URL, `customIcon`, and `background` URL | No |
| `PUT /api/site` | Set `title`; optional `icon` data URI, where `""` restores the default and omission keeps it | Yes |
| `PUT /api/site/background` | Replace the custom background with a raw JPEG, PNG, GIF, or WebP body | Yes |
| `DELETE /api/site/background` | Remove the custom background | Yes |
| `GET /site-icon` | Serve the custom page icon | No |
| `GET /backgrounds/{name}` | Serve the current custom background; old names return 404 | No |

Editing access is granted to everyone when no password is configured. Requests
with JSON bodies require `Content-Type: application/json`; the background upload
body is the image itself. All non-GET/HEAD
requests require `X-Harbor-Request: 1`; cross-origin browser changes are rejected.
JSON decoding rejects unknown fields, trailing objects, and oversized bodies.
Errors use a JSON `error` string. The browser never needs cross-origin API access.

## State and Failure Boundaries

- Server state owns service definitions, home visibility, and the page title,
  icon, and custom background image; browser state owns theme, the selected
  wallpaper, clock format, open panels, and unsaved form values.
- Mutations validate first, persist a replacement snapshot, then expose the
  updated collection in memory. Failed writes retain the previous collection.
- The visibility endpoint changes only `hidden`, avoiding replacement of other
  fields when toggling a card from All apps.
- Icon fetching is an explicit, bounded request. Failures do not prevent saving
  a service without an icon. No service status is inferred from icon availability.
- Stored icons and backgrounds are served with a sandbox policy. Background
  URLs change with each upload and are cached as immutable.
- Sessions live only in memory. The API's public collection is intentional.

See [data model](data-model.md), [secure development](../development/security.md),
and the [design contract](../../DESIGN.md).
