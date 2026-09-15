import assert from "node:assert/strict";
import { execFileSync } from "node:child_process";
import { randomBytes } from "node:crypto";
import { setTimeout as delay } from "node:timers/promises";

const image = process.argv[2] || "harbor:check";
const docker = process.argv[3] || "docker";
const name = `harbor-smoke-${randomBytes(6).toString("hex")}`;
const password = randomBytes(24).toString("hex");
let baseURL;
let cookie = "";
let cleaned = false;
let phase = "container startup";

function command(args, timeout = 60000) {
  return execFileSync(docker, args, { encoding: "utf8", timeout, stdio: ["ignore", "pipe", "pipe"] }).trim();
}

function cleanup() {
  if (cleaned) return;
  cleaned = true;
  try { command(["rm", "-f", "-v", name], 30000); }
  catch {
    console.error(`Could not remove ${name}; inspect and remove this smoke container and its anonymous volume.`);
    process.exitCode = 1;
  }
}

for (const signal of ["SIGINT", "SIGTERM"]) {
  process.on(signal, () => { cleanup(); process.exit(1); });
}

async function request(path, method = "GET", body) {
  return fetch(`${baseURL}${path}`, {
    method,
    headers: { "Content-Type": "application/json", "X-Harbor-Request": "1", Cookie: cookie },
    body: body === undefined ? undefined : JSON.stringify(body),
    signal: AbortSignal.timeout(5000),
    redirect: "error",
  });
}

function resolveAddress() {
  const binding = command(["port", name, "8080/tcp"]);
  assert.match(binding, /^127\.0\.0\.1:\d+$/);
  baseURL = `http://${binding}`;
}

async function ready() {
  const deadline = Date.now() + 30000;
  while (Date.now() < deadline) {
    try {
      const response = await request("/healthz");
      if (response.ok && (await response.text()).trim() === "ok") return;
    } catch { /* Startup and restart can briefly refuse a connection. */ }
    await delay(250);
  }
  throw new Error("The isolated container did not become ready within 30 seconds.");
}

async function login() {
  const response = await request("/api/login", "POST", { password });
  assert.equal(response.status, 200);
  cookie = response.headers.get("set-cookie")?.split(";")[0] || "";
  assert.ok(cookie.startsWith("harbor_session="));
  await response.json();
}

try {
  command(["run", "-d", "--name", name, "--pull=never", "-p", "127.0.0.1::8080", "--read-only", "--cap-drop", "ALL", "--security-opt", "no-new-privileges:true", "-e", `HARBOR_ADMIN_PASSWORD=${password}`, image]);
  resolveAddress();
  await ready();
  phase = "health check and embedded assets";
  command(["exec", name, "/harbor", "healthcheck"]);

  for (const [path, contentType] of [["/", "text/html"], ["/app.js", "javascript"], ["/style.css", "text/css"], ["/wallpaper.jpg", "image/jpeg"]]) {
    const response = await request(path);
    assert.equal(response.status, 200, path);
    assert.ok(response.headers.get("content-type").includes(contentType), path);
    assert.ok((await response.arrayBuffer()).byteLength > 0, path);
  }
  assert.deepEqual(await (await request("/api/services")).json(), []);
  phase = "authentication and service creation";
  const item = { name: "Smoke media", url: "https://media.example/", description: "Synthetic service", icon: "", hidden: false };
  const denied = await request("/api/services", "POST", item);
  assert.equal(denied.status, 401);
  await denied.json();
  await login();
  const created = await request("/api/services", "POST", item);
  assert.equal(created.status, 201);
  const service = await created.json();
  assert.ok(service.id);
  const hidden = await request(`/api/services/${service.id}/visibility`, "PATCH", { hidden: true });
  assert.equal(hidden.status, 200);
  assert.equal((await hidden.json()).hidden, true);

  phase = "container restart";
  command(["restart", name]);
  // Docker can assign a different ephemeral host port after a restart.
  resolveAddress();
  await ready();
  phase = "restart persistence and deletion";
  assert.equal((await (await request("/api/session")).json()).canEdit, false);
  const saved = await (await request("/api/services")).json();
  assert.equal(saved.length, 1);
  assert.deepEqual(saved[0], { ...item, id: service.id, hidden: true });

  await login();
  const removed = await request(`/api/services/${service.id}`, "DELETE");
  assert.equal(removed.status, 204);
  assert.deepEqual(await (await request("/api/services")).json(), []);
  console.log("Docker smoke passed: assets, protected writes, CRUD, home visibility, restart persistence, and session invalidation.");
} catch (error) {
  // Child-process errors can contain the generated editing secret in argv.
  console.error(`Docker smoke failed during ${phase}: ${error.message.replaceAll(password, "[redacted]").slice(0, 1200)}`);
  try { console.error(command(["logs", "--tail", "20", name], 10000)); }
  catch { console.error("Container logs are unavailable."); }
  process.exitCode = 1;
} finally {
  cleanup();
}
