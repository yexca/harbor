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
3. Commit through the configured signing setup and verify CI for that commit.
4. Configure the registry credentials below, then push the matching version tag
   only as part of an authorized release. Pushing the tag starts image publication.

## GitHub Actions

`ci.yml` runs on pull requests, pushes to `main`, and manual dispatch. It is also
reusable by `release.yml`. CI runs `make ci` (including race and isolated Docker
smoke tests) and `make sensitive-check`, with read-only repository permissions.

`release.yml` runs on pushed `v*` tags. Preparation requires a stable
`v<major>.<minor>.<patch>` tag exactly matching `VERSION`; malformed tags,
prereleases, and invalid registry configuration fail before publishing. The same
tagged source must pass the reusable CI workflow before registry login and build.
Third-party actions are pinned to commit SHAs.

Docker Hub settings are fixed in the workflow: login username `yexca` and image
`yexca/harbor`. No repository variables are needed. Configure the following in
**Settings → Secrets and variables → Actions → Secrets**:

| Kind | Name | Value |
| --- | --- | --- |
| Repository secret | `DOCKERHUB_TOKEN` | Docker Hub access token for `yexca` with permission to push to `yexca/harbor` |

Create the `yexca/harbor` Docker Hub repository and grant the token access before
the first tag. To publish a fork to a different Docker Hub account, edit the two
constants in the workflow's `env` block.
Keep the token in **Secrets**, not a plaintext Actions variable or source file.

GHCR uses the automatic `GITHUB_TOKEN` with `packages: write` only in the publish
job. The destination is `ghcr.io/<owner>/<repository>`, derived from the GitHub
repository name and converted to lowercase. Existing packages must grant this
repository write access. Configure package visibility separately if anonymous
pulls should be allowed; publishing does not change that setting.

One Buildx build produces `linux/amd64` and `linux/arm64` images and pushes the
same tags to GHCR and Docker Hub. A `v1.2.3` release produces `:v1.2.3` and
`:latest` in both registries. Releases are serialized; `latest` means the most
recently published stable tag, not necessarily the highest version number.
Prefer the version tag or the published digest when pinning deployments.

The workflow summary records image references and the digest. Registry pushes
are not atomic across both services: if one registry fails, inspect both before
rerunning the failed release workflow. No images are published for ordinary
branch pushes or pull requests. The workflow publishes container images only;
it does not create GitHub Release notes or upload standalone executables.

A local build or version file is not evidence of a published release. Confirm
the Actions run and both registry manifests after an authorized tag push.

After successful image publication, publish the GitHub Release separately using
the existing tag and its archived release notes. For example, with GitHub CLI:

```sh
gh release create v0.1.0 --verify-tag --title v0.1.0 --notes-file docs/history/v0.1.0.md --latest
```

Use the version being released in each argument. Do not move an existing release
tag to include later changes.
