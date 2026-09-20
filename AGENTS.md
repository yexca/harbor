# Agent Guide

## Read First

- [README](README.md) and [documentation map](docs/README.md).
- [Overview](docs/overview.md) and [architecture](docs/architecture/index.md).
- [Data model](docs/architecture/data-model.md).
- [Design contract](DESIGN.md) before changing the interface.
- [Secure development](docs/development/security.md) before changing requests,
  authentication, icons, or persistence.

## Product Boundaries

Harbor is a lightweight NAS start page. Preserve these constraints:

- English UI, errors, code comments, and documentation. No i18n layer yet.
- A wallpaper, clock/date, and home service cards are the primary screen.
- Settings stays in the top-right corner. Management belongs in panels.
- All apps opens through right-click, the bottom launcher, Settings, or `/`.
  Keep the final add tile and an accessible touch/keyboard alternative.
- Each service has one HTTP(S) address. Home visibility is presentation only;
  hidden services remain in All apps and the public API.
- No automatic LAN/WAN switching, health polling, service discovery, media
  proxying, multi-user accounts, or synchronization service in the current scope.
- Keep the runtime to one Go process with embedded HTML/CSS/JavaScript and local
  assets. Prefer the standard library. Introduce a dependency or build system
  only to solve an explicit requirement that the existing stack cannot reasonably handle.

## Implementation Rules

- `server/main.go` composes HTTP, validation, and sessions; `server/store.go` owns
  persistence; `server/icons.go` owns outbound requests and image validation;
  `server/web/` owns browser UI. Keep `go.mod` at the repository root and run
  the application from there with `make run` or `go run ./server`.
  Extract modules when an actual workflow warrants it, not for speculative layering.
- One process owns one data directory. Keep atomic writes in the same directory,
  persist before updating memory, and preserve the previous state on failure.
- Preserve existing configurations, including omitted `hidden` fields. A data
  format change needs an explicit compatibility path and regression coverage.
- Reuse the bounded icon transport. Private LAN targets are intentional; reject
  credentials and forbidden addresses, validate redirects, and dial validated IPs.
- Never render service text as HTML or load service icons from remote URLs in
  the browser. Keep icons validated and stored locally.
- Keep appearance preferences in browser storage and service/home selections
  in the NAS store. Never imply that hiding a card restricts access.
- Maintain keyboard focus, accessible labels, touch access, responsive layouts,
  and reduced-motion support. See [DESIGN.md](DESIGN.md).

## Validation

The [Makefile](Makefile) is the canonical entry point. Use the smallest sufficient target:

- Documentation: `make docs-check`.
- Go behavior: `make backend-check`; concurrency: `make test-race`.
- Browser source: `make web-check`, then manually exercise the affected flow.
- Repository scripts: `make scripts-test`.
- Docker/runtime: `make docker-smoke`, which builds and tests an isolated container.
- Portable aggregate: `make ci-local`. Full Linux CI equivalent: `make ci`.

Before every commit, run `make sensitive-check` and inspect the staged diff.
The scan checks the index as well as non-ignored working files; it is a heuristic,
not proof that a change contains no private information. See
[testing](docs/development/testing.md) for limits and commands.

Tests should protect behavior, state transitions, or a regression. Do not add
snapshot or implementation-mirroring tests for routine prose or visual changes.
Use reserved example domains and documentation IP ranges for incidental fixtures.
Use temporary directories and isolated containers; never test against saved NAS data.

## Documentation and Handoff

- Update the user, architecture, operations, or development page that owns a change.
- Record durable decisions in `docs/decisions/`; release summaries in `docs/history/`.
- [VERSION](VERSION) is release metadata; JSON schema version is independent.
  Version tags trigger validated Docker publishing; see the release guide for
  required registry variables and secrets. Do not push tags without release authorization.
- Use English Conventional Commit messages, for example
  `fix(icons): preserve the previous icon after a failed fetch`.
- Respect configured commit signing. Do not disable signing, replace a signer,
  or use a no-sign flag to bypass an unavailable signing agent.
- Keep runtime data, backups, credentials, personal endpoints/paths, screenshots
  of real collections, and local test artifacts out of Git.
- Report what changed, which checks ran, and any checks that could not run.
