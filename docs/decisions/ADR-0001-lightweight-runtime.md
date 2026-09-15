# ADR-0001: Lightweight Single-Process Runtime

Status: Accepted

## Context

Harbor is a small personal NAS launcher. It needs persistent editing and icon
fetching but should remain inexpensive to run and easy to deploy with Docker.

## Decision

Use Go's standard library, embedded native HTML/CSS/JavaScript, and a single
JSON file. Package one process in a small non-root container. Keep all presentation
assets local. Development scripts may use Node's built-in modules without npm
dependencies; Node is not part of the runtime.

## Consequences

There is no frontend installation/build pipeline or external database service.
The store is simple to back up, and all assets ship with the executable.

Whole-collection reads/writes and inline icons grow with the collection. The
store is limited to 500 services and one process per data directory. Static asset
changes require recompilation. Add a database or frontend framework only when a
concrete new requirement justifies revisiting this decision.
