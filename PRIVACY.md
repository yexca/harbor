# Privacy and Data Handling

Harbor has no analytics, telemetry, advertising, remote fonts, or hosted icon
lookup. Its assets are embedded and served by the local application.

## Stored Data

- The NAS stores service IDs, names, addresses, descriptions, icons, and home
  visibility in `services.json`. This file is not encrypted by Harbor.
- The browser stores theme, wallpaper, and clock-format preferences in localStorage.
  These preferences do not synchronize between devices.
- An optional editing password is supplied through the process environment.
  It is not written into `services.json`.
- Editing sessions are held in server memory for up to 24 hours and use an
  HttpOnly cookie. Restarting the server invalidates them.
- Server logs include startup configuration paths and operational errors. Treat
  logs and backups as private deployment data.

## Network Traffic

Opening a card navigates the browser to its configured service. Auto-fetch and
direct image fetch contact the requested website from the Harbor server, may
follow redirects and declared icon URLs, and store the resulting icon locally.
Those websites can observe the server's network address and the requested URL.
Harbor does not forward browser sessions or configured service credentials.

There is no periodic service polling. Icon fetching happens only when requested.
Links must be reachable from the user's device; successful server-side icon
fetching does not establish that reachability.

## Visibility and Removal

Anyone who can reach Harbor can read every saved service, including hidden cards.
Use reverse-proxy access control to restrict the collection itself.

Deleting a service removes it and its stored icon from the current configuration.
Old copies can remain in backups. Clearing browser site data removes local
appearance preferences. Operators control backup retention and data-directory
removal; Harbor does not provide remote deletion or a hosted account service.
