# Commit and Release

## Commits

The primary branch is `main`. Use focused branches and English Conventional
Commits in the form `<type>(scope): <description>`.

Run the appropriate Makefile checks, then `make sensitive-check`. Review
`git diff --cached --check`, `git diff --cached --stat`, and the complete staged
diff before committing. Do not stage local data, backups, environment files,
build output, or real service details.

Use the existing Git signing configuration. If its signing agent is unavailable,
restore access to that agent instead of disabling signing or replacing the signer.
Commit identity, signing keys, and remotes are local configuration; do not copy
them from another project into tracked files.

## Versions and History

[VERSION](../../VERSION) contains the intended application version using
`v<major>.<minor>.<patch>`. It is release metadata, not a runtime version endpoint
or the JSON schema version. The initial value is an unreleased baseline until a
matching tag is deliberately created.

Keep upcoming changes in [unreleased notes](../history/unreleased.md). Before
tagging, move the release body to `docs/history/<tag>.md`, link it from the
[history index](../history/index.md), and start a new unreleased page. Do not
repeat the version as a level-one heading in the release body.

## Release Preparation

1. Review VERSION, public docs, data compatibility, asset provenance, and release notes.
2. Run `make ci` and `make sensitive-check` on the exact release source.
3. Commit through the configured signing setup and verify CI for that commit
   after a remote has been configured.
4. Create the matching tag and publish artifacts only as part of an authorized release.

This repository currently has validation CI only. No remote URL, registry,
automatic image publishing, security reporting endpoint, or distribution license
is inferred from another project. Establish these before public distribution.
Do not claim a release exists merely because VERSION or an image tag exists locally.
