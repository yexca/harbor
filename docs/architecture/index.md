# Architecture

```text
Browser -> same-origin HTTP API -> JSON store
                               -> icon fetcher -> website
```

## Modules

| Location | Responsibility |
| --- | --- |
| `server/main.go` | Server lifecycle, routes, request validation, sessions, security headers, embedded assets |
| `server/store.go` | Service identity, collection limits, synchronized reads/writes, atomic JSON replacement |
| `server/icons.go` | URL/address policy, bounded icon discovery, image validation and encoding |
| `server/web/index.html` | Home screen, All apps, Settings, editor, login, and confirmation dialogs |
| `server/web/style.css` | Wallpaper, glass surfaces, themes, responsive grid, focus and motion behavior |
| `server/web/app.js` | Fetch API client, rendering, form state, dialogs, clock, browser preferences |
| `server/main_test.go` | HTTP, store, authentication, icon, and compatibility regression tests |
| `scripts/` | Development-only repository checks and isolated container smoke testing |

Application source and assets live in `server/`, with the single Go module
declared by the root `go.mod`. The Go files remain one `package main`.

The frontend has no build step. The `web/*` embed pattern in `server/main.go`
includes `server/web/` at compile time. Public asset URLs remain `/app.js`,
`/style.css`, `/favicon.svg`, and `/wallpaper.jpg`. A running process does not
pick up edited assets until it is rebuilt/restarted.

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

Editing access is granted to everyone when no password is configured. Requests
with JSON bodies require `Content-Type: application/json`. All non-GET/HEAD
requests require `X-Harbor-Request: 1`; cross-origin browser changes are rejected.
JSON decoding rejects unknown fields, trailing objects, and oversized bodies.
Errors use a JSON `error` string. The browser never needs cross-origin API access.

## State and Failure Boundaries

- Server state owns service definitions and home visibility; browser state owns
  theme, wallpaper, clock format, open panels, and unsaved form values.
- Mutations validate first, persist a replacement snapshot, then expose the
  updated collection in memory. Failed writes retain the previous collection.
- The visibility endpoint changes only `hidden`, avoiding replacement of other
  fields when toggling a card from All apps.
- Icon fetching is an explicit, bounded request. Failures do not prevent saving
  a service without an icon. No service status is inferred from icon availability.
- Sessions live only in memory. The API's public collection is intentional.

See [data model](data-model.md), [secure development](../development/security.md),
and the [design contract](../../DESIGN.md).
