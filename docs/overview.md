# Overview

Harbor is a self-hosted start page for services such as Emby and Komga. It opens
with a background, local clock/date, and a grid of selected service cards. A
separate All apps panel contains the complete collection and editing controls.

## Current Scope

- Add, edit, and delete services with a name, one address, description, and icon.
- Fetch or upload icons and store them with the collection.
- Select which services appear on home without deleting them.
- Search All apps and customize local appearance and clock preferences.
- Optionally protect editing with one administrator password.
- Run one lightweight container with persistent JSON data.

The UI, source comments, and documentation are English. There is no i18n system,
automatic LAN/WAN switching, multi-address routing, health polling, container
discovery, media proxy, multi-user account model, or cloud synchronization.

## Deployment Model

A single Go process serves embedded browser assets and a same-origin JSON API.
It uses the Go standard library, local assets, and a single data directory.
No database server, Node.js runtime, frontend bundler, or external CDN is needed
to run Harbor. Developer checks use Node.js without npm packages.

One process owns each data directory. The collection is readable by every
visitor; the optional password controls changes. Appearance belongs to each
browser, while service definitions and home visibility are shared on the NAS.

See the [user guide](user/index.md), [architecture](architecture/index.md), and
[lightweight architecture decision](decisions/ADR-0001-lightweight-runtime.md).
