import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import { existsSync, mkdtempSync, readFileSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { fileURLToPath } from "node:url";
import test from "node:test";
import { releaseMetadata } from "./release.mjs";

const release = { refType: "tag", tag: "v1.2.3", version: "v1.2.3\r\n", repository: "Example/Harbor", username: "example", image: "example/harbor" };

test("Release publishes the same stable version and latest to both registries", () => {
  assert.deepEqual(releaseMetadata(release), {
    version: "v1.2.3",
    tags: ["ghcr.io/example/harbor:v1.2.3", "ghcr.io/example/harbor:latest", "docker.io/example/harbor:v1.2.3", "docker.io/example/harbor:latest"],
  });
  assert.ok(releaseMetadata({ ...release, image: "example-team/harbor" }).tags.includes("docker.io/example-team/harbor:v1.2.3"));
});

test("Branches, malformed tags, prereleases, and mismatched VERSION cannot publish", () => {
  assert.throws(() => releaseMetadata({ ...release, refType: "branch" }), /stable/);
  for (const tag of ["main", "v1.2", "v01.2.3", "v1.2.3-rc.1", "v1.2.3\ninjected", "v1.2.3\n", ""]) {
    assert.throws(() => releaseMetadata({ ...release, tag }), /stable/);
  }
  assert.throws(() => releaseMetadata({ ...release, version: "v1.2.4" }), /match VERSION/);
});

test("Registry configuration cannot add tags, credentials, or output lines", () => {
  for (const image of [undefined, "", "example/Harbor", "example/harbor:latest", "docker.io/example/harbor", "example/harbor\nextra", "example/harbor\n", "user:password@example/harbor"]) {
    assert.throws(() => releaseMetadata({ ...release, image }), /DOCKERHUB_IMAGE/);
  }
  for (const username of [undefined, "", "example\nextra", "user@example"]) {
    assert.throws(() => releaseMetadata({ ...release, username }), /DOCKERHUB_USERNAME/);
  }
  assert.throws(() => releaseMetadata({ ...release, repository: "example/harbor\nextra" }), /GITHUB_REPOSITORY/);
});

test("Release CLI returns Actions outputs only after validation succeeds", (t) => {
  const dir = mkdtempSync(join(tmpdir(), "harbor-release-test-"));
  t.after(() => rmSync(dir, { recursive: true, force: true }));
  const version = readFileSync(new URL("../VERSION", import.meta.url), "utf8").trim();
  const output = join(dir, "output");
  const env = { ...process.env, GITHUB_REF_TYPE: "tag", GITHUB_REF_NAME: version, GITHUB_REPOSITORY: release.repository, DOCKERHUB_USERNAME: release.username, DOCKERHUB_IMAGE: release.image, GITHUB_OUTPUT: output };
  const script = fileURLToPath(new URL("release.mjs", import.meta.url));
  const result = spawnSync(process.execPath, [script], { env, encoding: "utf8" });
  assert.equal(result.status, 0, result.stderr);
  const expected = releaseMetadata({ ...release, version, tag: version });
  assert.equal(readFileSync(output, "utf8"), `version=${version}\ntags<<HARBOR_TAGS\n${expected.tags.join("\n")}\nHARBOR_TAGS\n`);
  const rejected = join(dir, "rejected");
  const failed = spawnSync(process.execPath, [script], { env: { ...env, GITHUB_REF_TYPE: "branch", GITHUB_OUTPUT: rejected }, encoding: "utf8" });
  assert.notEqual(failed.status, 0);
  assert.equal(existsSync(rejected), false);
});
