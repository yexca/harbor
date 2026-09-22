# ADR-0003: Shared Site Identity and Custom Background

Status: Accepted

## Context

Operators want Harbor to carry their own name, tab icon, and background image.
A title and favicon describe the deployment rather than one browser, and an
uploaded photo can exceed what browser storage holds reliably.

## Decision

Store the title, page icon, and one custom background image on the NAS, in
`site.json` and a store-named `background-*` file beside `services.json`. The
server renders the title, icon URL, and background URL into the page. Editing
them requires the same editing access as services.

Keep which background a device displays in browser storage, as ADR-0002
decided. The uploaded image becomes a fourth **Custom** choice, selected
automatically on the uploading device.

## Consequences

Existing data directories keep working: an absent `site.json` means defaults.
Backups should include the whole data directory to keep customization. The
title, icon, and image are public to anyone who can reach Harbor. Devices do not
switch to a new background without their own choice.
