import { execFileSync } from "node:child_process";
import { existsSync, readFileSync, statSync } from "node:fs";
import { dirname, isAbsolute, relative, resolve, sep } from "node:path";
import { fileURLToPath } from "node:url";

export const root = resolve(dirname(fileURLToPath(import.meta.url)), "..");

function git(args, cwd = root) {
  return execFileSync("git", args, { cwd, encoding: "utf8", maxBuffer: 32 * 1024 * 1024 });
}

export function repositoryFiles(cwd = root) {
  return [...new Set(git(["ls-files", "--cached", "--others", "--exclude-standard", "-z"], cwd).split("\0").filter(Boolean))].sort();
}

function markdownBody(text) {
  return text.replace(/^([ \t]*)(`{3,}|~{3,})[^\n]*\n[\s\S]*?^\1\2[^\n]*$/gm, "");
}

export function markdownErrors(file, text, cwd = root) {
  const errors = [];
  const body = markdownBody(text);
  const links = [...body.matchAll(/!?\[[^\]\n]*\]\(\s*(<[^>]+>|[^\s)]+)(?:\s+"[^"]*")?\s*\)/g)].map((match) => match[1]);
  links.push(...[...body.matchAll(/^\s*\[[^\]]+\]:\s*(<[^>]+>|\S+)/gm)].map((match) => match[1]));
  for (let link of links) {
    link = link.replace(/^<|>$/g, "");
    if (/^(?:https?:|mailto:)/i.test(link)) continue;
    if (/^[a-z][a-z\d+.-]*:/i.test(link) || link.startsWith("//")) {
      errors.push("Use a repository-relative documentation link.");
      continue;
    }
    let path;
    try { path = decodeURIComponent(link.split(/[?#]/)[0]); }
    catch { errors.push("Malformed URL encoding in a local link."); continue; }
    const target = path ? resolve(dirname(resolve(cwd, file)), path) : resolve(cwd, file);
    const local = relative(cwd, target);
    if (local === ".." || local.startsWith(`..${sep}`) || isAbsolute(local)) {
      errors.push("A documentation link leaves the repository.");
    } else if (!existsSync(target)) {
      errors.push(`Missing local link target: ${path}`);
    }
  }
  return errors;
}

const sensitivePath = /(^|\/)(?:data|backups|\.tmp|bin|dist|coverage|node_modules)(?:\/|$)|(^|\/)\.env(?:\.(?!example$).*)?$|(^|\/)services(?:\.backup[^/]*)?\.json$|\.(?:pem|key|log|db|sqlite(?:3)?|exe)$|\.db-(?:wal|shm)$/i;

function exampleHost(host) {
  return /^(?:localhost|127\.0\.0\.1|\[::1\]|example\.(?:com|org|net))$/i.test(host)
    || /\.(?:example|test|invalid|localhost)$/.test(host)
    || /\.example\.(?:com|org|net)$/.test(host)
    || /^(?:192\.0\.2|198\.51\.100|203\.0\.113)\.\d+$/.test(host)
    || /^\[2001:db8:/i.test(host);
}

export function sensitiveFindings(file, text, allowlist = []) {
  const findings = [];
  if (sensitivePath.test(file)) findings.push({ line: 1, rule: "runtime data, secret file, or generated artifact" });
  if (text.includes("\0")) return findings;
  text.split(/\r?\n/).forEach((line, index) => {
    const add = (rule) => findings.push({ line: index + 1, rule });
    if (/-----BEGIN (?:[A-Z]+ )?PRIVATE KEY-----|\bgh[pousr]_[A-Za-z0-9]{20,}\b|\bgithub_pat_[A-Za-z0-9_]{30,}\b|\bAKIA[A-Z0-9]{16}\b/.test(line)) add("possible credential");
    if (/[a-z]:[\\/](?:Users|Documents and Settings)[\\/]|\/(?:Users|home)\/[^\s/]+\//i.test(line)) add("personal absolute path");
    for (const match of line.matchAll(/\b(?:HARBOR_ADMIN_PASSWORD|API_KEY|ACCESS_TOKEN|CLIENT_SECRET)\s*[=:]\s*["']?([^\s"'`;},]+)/g)) {
      if (!/^(?:replace[-_]|example|change[-_]|\$|process\.|os\.)/i.test(match[1])) add("non-placeholder secret assignment");
    }
    for (const match of line.matchAll(/https?:\/\/[^\s<>"'`\\]+/g)) {
      const raw = match[0].replace(/[.,);*\]}]+$/, "");
      if (raw.includes("${")) continue;
      let url;
      try { url = new URL(raw); } catch { continue; }
      if ((url.username || url.password) && !(file.endsWith("_test.go") && exampleHost(url.hostname))) add("credentials embedded in URL");
      if (exampleHost(url.hostname)) continue;
      if (!allowlist.some((entry) => entry.url === raw && entry.files.includes(file))) add("non-example URL requires review and an exact file-scoped allowlist entry");
    }
  });
  return findings;
}

export function scanRepository(cwd = root, allowlist = []) {
  const findings = [];
  const staged = new Set(git(["ls-files", "--cached", "-z"], cwd).split("\0").filter(Boolean));
  for (const file of repositoryFiles(cwd)) {
    const snapshots = [];
    if (staged.has(file)) snapshots.push(["index", git(["show", `:${file}`], cwd)]);
    const path = resolve(cwd, file);
    if (existsSync(path) && statSync(path).isFile()) snapshots.push(["working tree", readFileSync(path, "utf8")]);
    for (const [snapshot, text] of snapshots) {
      for (const finding of sensitiveFindings(file, text, allowlist)) findings.push({ file, snapshot, ...finding });
    }
  }
  return findings;
}

function run(command) {
  if (command === "help") {
    console.log(`Harbor repository commands
  make run / build          Run locally / build bin/harbor
  make format               Apply gofmt
  make backend-check        Check Go formatting, vet, and tests
  make web-check            Check browser JavaScript syntax
  make docs-check           Check local Markdown link targets
  make scripts-test         Test repository check behavior
  make sensitive-check      Scan index and non-ignored working files
  make ci-local             All portable checks and local build
  make ci                   Portable checks, race test, Docker smoke
  make docker-smoke         Build and test an isolated container
  make docker-up/down       Build/start or stop development Compose (preserves data)
  make docker-status/logs   Inspect development Compose
Overrides: GO, NODE, DOCKER, GO_PARALLEL, DOCKER_IMAGE, COMPOSE_FILE, COMPOSE_PROJECT`);
    return;
  }
  const files = repositoryFiles();
  if (command === "format" || command === "format-check") {
    const goFiles = files.filter((file) => file.endsWith(".go") && existsSync(resolve(root, file)));
    const result = goFiles.length ? execFileSync("gofmt", [command === "format" ? "-w" : "-l", ...goFiles], { cwd: root, encoding: "utf8" }) : "";
    if (result.trim()) throw new Error(`Go formatting is required:\n${result}`);
    console.log(command === "format" ? "Go files formatted." : "Go formatting passed.");
  } else if (command === "docs") {
    const errors = [];
    for (const file of files.filter((file) => file.endsWith(".md") && existsSync(resolve(root, file)))) {
      for (const error of markdownErrors(file, readFileSync(resolve(root, file), "utf8"))) errors.push(`${file}: ${error}`);
    }
    if (errors.length) throw new Error(errors.join("\n"));
    console.log("Local Markdown link targets passed (external URLs and heading fragments are not checked).");
  } else if (command === "sensitive") {
    const allowlist = JSON.parse(readFileSync(resolve(root, "scripts/privacy-allowlist.json"), "utf8"));
    if (!Array.isArray(allowlist) || allowlist.some((entry) => typeof entry.url !== "string" || !Array.isArray(entry.files) || !entry.files.length || !entry.reason?.trim())) throw new Error("Every allowlist entry needs an exact URL, owner files, and a reason.");
    const findings = scanRepository(root, allowlist);
    if (findings.length) throw new Error(findings.map((f) => `${f.file}:${f.line} (${f.snapshot}): ${f.rule}`).join("\n"));
    console.log("Sensitive-file scan passed for the index and non-ignored working files. Review the staged diff as well.");
  } else {
    throw new Error("Usage: node scripts/check.mjs help|format|format-check|docs|sensitive");
  }
}

if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  try { run(process.argv[2]); }
  catch (error) { console.error(error.message); process.exitCode = 1; }
}
