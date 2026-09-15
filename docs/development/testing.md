# Testing and Validation

The [Makefile](../../Makefile) is the canonical local and CI entry point. Choose
the smallest target that covers the change.

| Target | Scope |
| --- | --- |
| `make format` | Apply gofmt to repository Go files |
| `make format-check` | Fail on unformatted Go files without modifying them |
| `make test` | Go behavioral tests |
| `make test-race` | Go tests with the race detector; needs a compatible C compiler |
| `make vet` | Go static checks |
| `make backend-check` | Formatting, vet, and Go tests |
| `make web-check` | Browser JavaScript syntax; not a browser integration test |
| `make docs-check` | Existence and containment of local Markdown link targets |
| `make scripts-test` | Repository checker behavior, including staged/private data cases |
| `make sensitive-check` | Index and non-ignored working files, including prospective untracked files |
| `make ci-local` | Backend, web, docs, checker tests, and a compiled local build |
| `make docker-build` | Build the production image as `harbor:check` by default |
| `make docker-smoke` | Build and exercise an isolated production container |
| `make ci` | `ci-local`, race testing, and Docker smoke; the Linux Actions sequence |

GitHub Actions invokes `make ci` and `make sensitive-check` on `main` pushes and
pull requests. Local validation does not imply that a remote CI run has occurred.
There is no frontend package installation or production bundling step.

## Coverage and Browser Checks

Go tests cover CRUD across restarts, concurrent writes, write failures, invalid
configuration preservation, editing sessions, request validation, home visibility
compatibility, embedded assets, icon discovery, and unsafe image/address rejection.

For optional coverage, run `go test -cover ./...`. Add tests for observable
contracts or regressions, not line-by-line implementation details.

For UI changes, run `make web-check`, start an isolated preview, and exercise the
affected flow in a browser. Verify home/All apps visibility, add/edit/delete,
failed icon fetches, Settings preferences, keyboard dismissal, focus, and narrow
viewport scrolling as relevant. Syntax checking alone does not verify these flows.

## Docker Smoke Isolation

The smoke script creates a unique container with a dynamically assigned
loopback-only port and an anonymous `/data` volume. It verifies static assets,
password-protected writes, service creation/deletion, home visibility, persistence
across restart, and session invalidation. It never mounts deployment data or
contacts configured NAS services. The test service uses a reserved example URL.

Success, failure, and handled interruption remove the test container and its
anonymous volume. Forced process termination can prevent cleanup; inspect any
reported `harbor-smoke-*` resource before manually removing it.

## Documentation and Privacy Check Limits

The Markdown checker recognizes ordinary inline links/images and reference
definitions, skips fenced code, and checks local destinations. It does not fetch
external URLs, validate heading fragments, or implement a full Markdown parser.

The privacy scanner checks both staged content and existing non-ignored working
files. It flags common credential patterns, personal absolute paths, runtime
data filenames, and non-example HTTP(S) URLs. Binary content is not inspected
for embedded metadata. It prints locations and rule names instead of secrets.

Public references belong in [the exact URL allowlist](../../scripts/privacy-allowlist.json)
with owner files and a reason. It never suppresses credential/path findings.
Use reserved example domains or documentation IP ranges for ordinary fixtures;
literal prohibited addresses are appropriate only when testing address rejection.

This is a heuristic and can miss secrets or flag intentional examples. Review
the actual staged diff and every allowlist change before committing. Ignoring a
file does not make it safe to share.
