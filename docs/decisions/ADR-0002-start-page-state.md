# ADR-0002: Home Selection and Local Appearance

Status: Accepted

## Context

The primary screen should feel like a personal start page, while every saved
service remains accessible and manageable without cluttering home.

## Decision

Home contains a wallpaper, clock/date, and selected service cards. All apps is a
separate panel opened by right-click or accessible launcher alternatives, ending
with an add tile. Settings is in the top-right corner.

Store each service's `hidden` flag on the NAS. Missing flags default to `false`
for compatibility. Keep theme, background, and clock preferences in the browser.
Hiding a service changes presentation only; the full collection stays public.

## Consequences

Home selection is consistent across devices, while appearance can vary per
browser. Existing services remain visible after an upgrade. Access control must
remain separate from home visibility. Future work must preserve keyboard/touch
access to the panel instead of relying on right-click alone.
