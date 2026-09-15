# Contributing

Read [AGENTS.md](AGENTS.md), [DESIGN.md](DESIGN.md), and the relevant section of
the [documentation](docs/README.md) before changing Harbor.

Keep changes focused and maintain English across the application and docs.
Harbor deliberately uses Go's standard library and plain browser assets. Tooling
must not add a production runtime or frontend build requirement without a clear need.

## Workflow

1. Create a focused branch from `main`.
2. Implement the change and update the documentation that owns the behavior.
3. Run the appropriate [Makefile checks](docs/development/testing.md).
4. Run `make sensitive-check`, stage only intended files, and review
   `git diff --cached` and `git diff --cached --check`.
5. Commit through your configured signing setup.

Use English Conventional Commits:

```text
feat(home): add a new background option
fix(store): preserve saved services after a failed write
docs(docker): clarify volume ownership
```

Do not commit runtime data, real service endpoints, credentials, personal paths,
or generated build/test artifacts. Use `example.com`, `*.example`, and
documentation IP ranges for examples; use `localhost` only for local setup.

Local setup is in [development](docs/development/local-dev.md). Commit, version,
and release conventions are in [commit and release](docs/development/commit-and-release.md).
For vulnerabilities, follow [SECURITY.md](SECURITY.md).
