import assert from "node:assert/strict";
import { execFileSync } from "node:child_process";
import { mkdtempSync, mkdirSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import test from "node:test";
import { markdownErrors, scanRepository, sensitiveFindings } from "./check.mjs";

function workspace(t) {
  const dir = mkdtempSync(join(tmpdir(), "harbor-repo-test-"));
  t.after(() => rmSync(dir, { recursive: true, force: true }));
  return dir;
}

test("Markdown checks resolve nested and encoded links without treating code examples as links", (t) => {
  const dir = workspace(t);
  mkdirSync(join(dir, "docs"));
  writeFileSync(join(dir, "README.md"), "# Readme\n");
  writeFileSync(join(dir, "docs", "with spaces.md"), "# Page\n");
  assert.deepEqual(markdownErrors("docs/index.md", "[root](../README.md#readme) [space](with%20spaces.md)\n```md\n[example](missing.md)\n```", dir), []);
  assert.equal(markdownErrors("docs/index.md", "[missing](absent.md)", dir).length, 1);
  assert.equal(markdownErrors("docs/index.md", "[escape](../../outside.md)", dir).length, 1);
});

test("Privacy checks report risky files without printing their contents", () => {
  assert.equal(sensitiveFindings("data/services.json", "{}")[0].rule, "runtime data, secret file, or generated artifact");
  assert.equal(sensitiveFindings(".env.example", ["HARBOR_ADMIN_PASSWORD=", ""].join("\n")).length, 0);
  const credential = "gh" + "p_" + "A".repeat(30);
  const findings = sensitiveFindings("sample.txt", credential);
  assert.equal(findings.length, 1);
  assert.ok(!JSON.stringify(findings).includes(credential));
  assert.equal(sensitiveFindings("notes.md", "C:" + "/Users/" + "sample/private").length, 1);
});

test("URL allowlists are exact and file scoped and do not suppress credential findings", () => {
  const url = "https://" + "public-service.org/icon.png";
  const allowlist = [{ url, files: ["ASSETS.md"], reason: "Public asset" }];
  assert.equal(sensitiveFindings("ASSETS.md", url, allowlist).length, 0);
  assert.equal(sensitiveFindings("notes.md", url, allowlist).length, 1);
  assert.equal(sensitiveFindings("ASSETS.md", `${url}?private=true`, allowlist).length, 1);
  const privateURL = "http://" + "192.168.23.45:8080";
  assert.equal(sensitiveFindings("notes.md", privateURL).length, 1);
  assert.equal(sensitiveFindings("notes.md", "https://service.example/path").length, 0);
  const credentialURL = "https://" + "user:secret@service.example/";
  assert.ok(sensitiveFindings("notes.md", credentialURL).some((f) => f.rule === "credentials embedded in URL"));
});

test("Scan catches staged content even after the working copy is sanitized, and skips ignored files", (t) => {
  const dir = workspace(t);
  const git = (...args) => execFileSync("git", args, { cwd: dir, stdio: "pipe" });
  git("init", "-q");
  writeFileSync(join(dir, ".gitignore"), ".env\n");
  writeFileSync(join(dir, ".env"), "private local settings");
  const secret = "API_" + "KEY=" + "synthetic-secret-value";
  writeFileSync(join(dir, "notes.txt"), secret);
  git("add", "notes.txt");
  writeFileSync(join(dir, "notes.txt"), "Sanitized working copy\n");
  writeFileSync(join(dir, "untracked.txt"), secret);
  const findings = scanRepository(dir);
  assert.ok(findings.some((f) => f.file === "notes.txt" && f.snapshot === "index"));
  assert.ok(findings.some((f) => f.file === "untracked.txt" && f.snapshot === "working tree"));
  assert.ok(!findings.some((f) => f.file === ".env"));
  assert.ok(!findings.some((f) => f.file === "notes.txt" && f.snapshot === "working tree"));
});
