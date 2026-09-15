"use strict";

const $ = (selector) => document.querySelector(selector);
const state = { services: [], canEdit: false, protected: false, loaded: false, editing: null, icon: "", iconManual: false, iconRequest: 0, iconController: null, lastIconURL: "", saving: false, toastTimer: null, clockTimer: null, editorReturn: null, afterLogin: null, visibilityPending: new Set() };

function storageGet(key) { try { return localStorage.getItem(key); } catch { return null; } }
function storageSet(key, value) { try { localStorage.setItem(key, value); } catch { /* Preferences are optional when browser storage is disabled. */ } }

const preferences = { theme: storageGet("harbor-theme") || "system", wallpaper: storageGet("harbor-wallpaper") || "mountain", clock24: storageGet("harbor-clock-24") !== "false" };
const systemTheme = window.matchMedia("(prefers-color-scheme: dark)");
function applyTheme(choice = preferences.theme) {
  if (!["system", "light", "dark"].includes(choice)) choice = "system";
  preferences.theme = choice;
  document.documentElement.dataset.theme = choice === "system" ? (systemTheme.matches ? "dark" : "light") : choice;
  document.querySelectorAll("[data-theme-choice]").forEach((button) => button.setAttribute("aria-pressed", String(button.dataset.themeChoice === choice)));
}
function applyWallpaper(choice = preferences.wallpaper) {
  if (!["mountain", "dusk", "midnight"].includes(choice)) choice = "mountain";
  preferences.wallpaper = choice;
  document.documentElement.dataset.wallpaper = choice;
  document.querySelectorAll("[data-wallpaper-choice]").forEach((button) => button.setAttribute("aria-pressed", String(button.dataset.wallpaperChoice === choice)));
}
document.querySelectorAll("[data-theme-choice]").forEach((button) => button.addEventListener("click", () => { applyTheme(button.dataset.themeChoice); storageSet("harbor-theme", preferences.theme); }));
document.querySelectorAll("[data-wallpaper-choice]").forEach((button) => button.addEventListener("click", () => { applyWallpaper(button.dataset.wallpaperChoice); storageSet("harbor-wallpaper", preferences.wallpaper); }));
systemTheme.addEventListener("change", () => applyTheme());
applyTheme();
applyWallpaper();

function updateClock() {
  clearTimeout(state.clockTimer);
  const now = new Date();
  const use24Hours = preferences.clock24;
  const parts = new Intl.DateTimeFormat("en-US", { hour: "2-digit", minute: "2-digit", hourCycle: use24Hours ? "h23" : "h12" }).formatToParts(now);
  const value = (type) => parts.find((part) => part.type === type)?.value || "";
  const clock = $("#clock");
  clock.textContent = `${value("hour")}:${value("minute")}`;
  if (!use24Hours) {
    const period = document.createElement("span");
    period.className = "clock-period";
    period.textContent = value("dayPeriod");
    clock.append(period);
  }
  clock.dateTime = now.toISOString();
  clock.setAttribute("aria-label", `Current time: ${clock.textContent}`);
  $("#date").textContent = new Intl.DateTimeFormat("en-US", { weekday: "long", month: "long", day: "numeric" }).format(now);
  $("#clock-format").checked = use24Hours;
  // Update at minute boundaries and pause the timer in background tabs.
  if (!document.hidden) state.clockTimer = setTimeout(updateClock, 60000 - Date.now() % 60000 + 20);
}
$("#clock-format").addEventListener("change", (event) => { preferences.clock24 = event.target.checked; storageSet("harbor-clock-24", String(preferences.clock24)); updateClock(); });
document.addEventListener("visibilitychange", updateClock);
updateClock();

async function api(path, options = {}) {
  const response = await fetch(path, {
    ...options,
    headers: { "Content-Type": "application/json", "X-Harbor-Request": "1", ...options.headers },
  });
  if (response.status === 204) return null;
  const data = await response.json().catch(() => ({}));
  if (!response.ok) {
    if (response.status === 401 && path !== "/api/login") {
      state.canEdit = false;
      updateAccess();
      render();
    }
    throw new Error(data.error || `Request failed (${response.status}). Please try again.`);
  }
  return data;
}

function svg(name, className = "icon") {
  const element = document.createElementNS("http://www.w3.org/2000/svg", "svg");
  element.setAttribute("class", className);
  element.setAttribute("aria-hidden", "true");
  const use = document.createElementNS("http://www.w3.org/2000/svg", "use");
  use.setAttribute("href", `#i-${name}`);
  element.append(use);
  return element;
}

function initials(name) {
  const words = name.trim().split(/\s+/).filter(Boolean);
  return (words.length > 1 ? [...words[0]][0] + [...words[1]][0] : [...(words[0] || "H")].slice(0, 2).join("")).toUpperCase();
}

function setIcon(element, name, icon) {
  element.replaceChildren();
  element.classList.toggle("has-image", Boolean(icon));
  element.dataset.color = String([...name].reduce((sum, char) => sum + char.codePointAt(0), 0) % 5);
  if (icon) {
    const image = document.createElement("img");
    image.src = icon;
    image.alt = "";
    image.decoding = "async";
    image.addEventListener("error", () => setIcon(element, name, ""), { once: true });
    element.append(image);
  } else {
    element.textContent = initials(name);
  }
}

function serviceCard(item, home = false) {
  const article = document.createElement("article");
  article.className = home ? "home-card" : "library-card";
  article.dataset.serviceId = item.id;
  const link = document.createElement("a");
  link.className = "card-link";
  link.href = item.url;
  link.target = "_blank";
  link.rel = "noopener noreferrer";
  link.setAttribute("aria-label", `Open ${item.name} in a new tab`);
  const icon = document.createElement("div");
  icon.className = "service-icon";
  setIcon(icon, item.name, item.icon);
  const copy = document.createElement("div");
  copy.className = "card-copy";
  const name = document.createElement("h2");
  name.className = "card-title";
  name.textContent = item.name;
  name.title = item.name;
  const description = document.createElement("p");
  description.className = "card-description";
  let host = item.url;
  try { host = new URL(item.url).host; } catch { /* Saved URLs are validated by the server. */ }
  description.textContent = home ? (item.description || host) : host;
  description.title = item.description || item.url;
  copy.append(name, description);
  link.append(icon, copy);
  article.append(link);
  if (!home && state.canEdit) {
    const actions = document.createElement("div");
    actions.className = "card-actions";
    const pin = document.createElement("button");
    pin.type = "button";
    pin.className = "home-pin";
    pin.dataset.homeId = item.id;
    pin.setAttribute("aria-label", `${item.hidden ? "Show" : "Hide"} ${item.name} ${item.hidden ? "on" : "from"} home`);
    pin.setAttribute("aria-pressed", String(!item.hidden));
    pin.title = item.hidden ? "Show on home" : "Hide from home";
    pin.disabled = state.visibilityPending.has(item.id);
    const label = document.createElement("span");
    label.textContent = item.hidden ? "Show on home" : "On home";
    pin.append(svg("home"), label);
    pin.addEventListener("click", () => toggleHome(item.id));
    const edit = document.createElement("button");
    edit.type = "button";
    edit.className = "icon-button edit-card";
    edit.setAttribute("aria-label", `Edit ${item.name}`);
    edit.title = `Edit ${item.name}`;
    edit.append(svg("edit"));
    edit.disabled = state.visibilityPending.has(item.id);
    edit.addEventListener("click", () => openEditor(item));
    actions.append(pin, edit);
    article.append(actions);
  }
  return article;
}

function render() {
  const query = $("#search").value.trim().toLowerCase();
  const filtered = state.services.filter((item) => `${item.name} ${item.description} ${item.url}`.toLowerCase().includes(query));
  const grid = $("#service-grid");
  grid.replaceChildren(...filtered.map((item) => serviceCard(item)));
  const home = state.services.filter((item) => !item.hidden);
  $("#home-grid").replaceChildren(...home.map((item) => serviceCard(item, true)));
  $("#service-count").textContent = state.services.length;
  $("#home-count").textContent = `${home.length} of ${state.services.length} on home`;
  $("#empty-home").hidden = !state.loaded || home.length > 0;
  $("#library-empty").hidden = !state.loaded || state.services.length > 0;
  $("#library-hint").textContent = state.canEdit ? "Choose which apps appear on your home screen." : "Unlock editing in Settings to organize your apps.";
  $("#no-results").hidden = !state.loaded || !state.services.length || filtered.length > 0;
  if (state.loaded) {
    const add = document.createElement("button");
    add.type = "button";
    add.className = "add-card";
    const circle = document.createElement("span");
    circle.className = "add-circle";
    circle.append(svg("plus"));
    const text = document.createElement("span");
    text.textContent = "Add a service";
    add.append(circle, text);
    add.addEventListener("click", () => openEditor());
    grid.append(add);
  }
}

function closeNavigationPanels() {
  $("#settings-dialog").close();
  $("#apps-dialog").close();
}

function openLibrary(focusSearch = false) {
  if ($("#service-dialog").open || $("#login-dialog").open || $("#delete-dialog").open) return;
  closeNavigationPanels();
  $("#search").value = "";
  render();
  $("#apps-dialog").showModal();
  if (focusSearch) $("#search").focus();
}

function openSettings() {
  closeNavigationPanels();
  $("#settings-dialog").showModal();
  $("#settings-button").setAttribute("aria-expanded", "true");
}

async function toggleHome(id) {
  const item = state.services.find((service) => service.id === id);
  if (!item || state.visibilityPending.has(id)) return;
  state.visibilityPending.add(id);
  render();
  try {
    const saved = await api(`/api/services/${id}/visibility`, { method: "PATCH", body: JSON.stringify({ hidden: !item.hidden }) });
    state.services = state.services.map((service) => service.id === id ? saved : service);
  } catch (error) {
    toast(error.message);
  } finally {
    state.visibilityPending.delete(id);
    render();
    if ($("#apps-dialog").open) [...document.querySelectorAll("[data-home-id]")].find((button) => button.dataset.homeId === id)?.focus();
  }
}

function updateAccess() {
  $("#access-button").hidden = !state.protected;
  $("#access-button span").textContent = state.canEdit ? "Lock editing" : "Unlock editing";
  $("#add-button").disabled = !state.loaded;
}

async function load() {
  $("#load-error").hidden = true;
  $("#loading").hidden = false;
  try {
    const [services, session] = await Promise.all([api("/api/services"), api("/api/session")]);
    state.services = services;
    state.canEdit = session.canEdit;
    state.protected = session.protected;
    state.loaded = true;
    updateAccess();
    render();
  } catch (error) {
    $("#load-error p").textContent = "Could not load your services. Check that Harbor is running, then try again.";
    $("#load-error").hidden = false;
  } finally {
    $("#loading").hidden = true;
  }
}

function showError(selector, message) { const element = $(selector); element.textContent = message; element.hidden = !message; }
function toast(message) {
  clearTimeout(state.toastTimer);
  $("#toast span").textContent = message;
  $("#toast").hidden = false;
  state.toastTimer = setTimeout(() => { $("#toast").hidden = true; }, 3500);
}

function openLogin() {
  closeNavigationPanels();
  $("#login-form").reset();
  showError("#login-error", "");
  $("#login-dialog").showModal();
  $("#password").focus();
}

function cancelIconRequest() {
  state.iconRequest++;
  state.iconController?.abort();
  state.iconController = null;
  $("#fetch-icon").disabled = false;
  $("#fetch-direct-icon").disabled = false;
}

function openEditor(item = null) {
  if (!state.canEdit) { state.afterLogin = () => openEditor(item); openLogin(); return; }
  state.editorReturn = $("#settings-dialog").open ? "settings" : "apps";
  closeNavigationPanels();
  cancelIconRequest();
  state.editing = item;
  state.icon = item?.icon || "";
  state.iconManual = Boolean(item?.icon);
  state.lastIconURL = item?.url || "";
  $("#service-form").reset();
  $("#service-name").value = item?.name || "";
  $("#service-url").value = item?.url || "";
  $("#service-description").value = item?.description || "";
  $("#service-on-home").checked = !item?.hidden;
  $("#dialog-title").textContent = item ? "Edit service" : "Add a service";
  $("#save-button").textContent = item ? "Save changes" : "Add service";
  $("#delete-button").hidden = !item;
  $("#icon-link-details").open = false;
  $("#icon-status").textContent = item?.icon ? "Your saved icon. Fetch or upload to replace it." : "An icon will be fetched when you enter an address.";
  showError("#form-error", "");
  updatePreview();
  $("#service-dialog").showModal();
  $("#service-name").focus();
}

function updatePreview() {
  setIcon($("#icon-preview"), $("#service-name").value || "Harbor", state.icon);
  $("#reset-icon").hidden = !state.icon;
}

function normalizeURL(raw) {
  raw = raw.trim();
  if (!raw) throw new Error("Enter a service address first.");
  if (!/^https?:\/\//i.test(raw)) {
    if (/^[a-z][a-z\d+.-]*:/i.test(raw) && !/^[^/:]+:\d+(?:\/|$)/.test(raw)) throw new Error("Use an http:// or https:// address.");
    const local = /^(localhost|[\d.]+|\[[\da-f:]+\]|[^/]+\.(local|lan|home|internal))(?::\d+)?(?:\/|$)/i.test(raw) || !raw.split("/")[0].includes(".");
    raw = `${local ? "http" : "https"}://${raw}`;
  }
  let url;
  try { url = new URL(raw); } catch { throw new Error("Enter a valid http:// or https:// address."); }
  if (!["http:", "https:"].includes(url.protocol) || !url.hostname || url.username || url.password) throw new Error("Use an http:// or https:// address without embedded credentials.");
  return url.href;
}

async function fetchIcon(direct = false, automatic = false) {
  if (automatic && state.iconManual) return;
  let url;
  try { url = normalizeURL($(direct ? "#icon-url" : "#service-url").value); }
  catch (error) { if (!automatic) $("#icon-status").textContent = error.message; return; }
  if (automatic && url === state.lastIconURL) return;
  cancelIconRequest();
  const request = state.iconRequest;
  const controller = new AbortController();
  state.iconController = controller;
  if (!direct) state.lastIconURL = url;
  $("#fetch-icon").disabled = true;
  $("#fetch-direct-icon").disabled = true;
  $("#icon-status").textContent = "Looking for an icon… You can still save this service.";
  try {
    const data = await api("/api/icon", { method: "POST", body: JSON.stringify({ url, direct }), signal: controller.signal });
    if (request !== state.iconRequest || !$("#service-dialog").open) return;
    state.icon = data.icon;
    state.iconManual = direct || !automatic;
    updatePreview();
    $("#icon-status").textContent = "Icon ready. It will be saved with your service.";
  } catch (error) {
    if (error.name !== "AbortError" && request === state.iconRequest) $("#icon-status").textContent = error.message;
  } finally {
    if (request === state.iconRequest) {
      $("#fetch-icon").disabled = false;
      $("#fetch-direct-icon").disabled = false;
      state.iconController = null;
    }
  }
}

$("#add-button").addEventListener("click", () => openEditor());
$("#settings-button").addEventListener("click", openSettings);
$("#open-apps").addEventListener("click", () => openLibrary());
$("#manage-apps").addEventListener("click", () => openLibrary());
$("#settings-dialog").addEventListener("close", () => $("#settings-button").setAttribute("aria-expanded", "false"));
$("#retry-button").addEventListener("click", load);
$("#search").addEventListener("input", render);
$("#clear-search").addEventListener("click", () => { $("#search").value = ""; render(); $("#search").focus(); });
$("#service-name").addEventListener("input", updatePreview);
$("#service-url").addEventListener("blur", () => fetchIcon(false, true));
$("#service-url").addEventListener("input", () => { if (state.iconController) { cancelIconRequest(); state.lastIconURL = ""; $("#icon-status").textContent = "Address changed. Leave the field to fetch its icon."; } });
$("#fetch-icon").addEventListener("click", () => fetchIcon());
$("#fetch-direct-icon").addEventListener("click", () => fetchIcon(true));
$("#upload-icon").addEventListener("click", () => $("#icon-file").click());
$("#reset-icon").addEventListener("click", () => {
  cancelIconRequest();
  state.icon = "";
  state.iconManual = true;
  updatePreview();
  $("#icon-status").textContent = "Using initials. You can fetch or upload an icon anytime.";
});
$("#icon-file").addEventListener("change", async (event) => {
  const file = event.target.files[0];
  if (!file) return;
  cancelIconRequest();
  const request = state.iconRequest;
  if (file.size > 512 * 1024) { $("#icon-status").textContent = "Choose an image smaller than 512 KB."; event.target.value = ""; return; }
  try {
    const data = await new Promise((resolve, reject) => {
      const reader = new FileReader();
      reader.onload = () => resolve(reader.result);
      reader.onerror = () => reject(new Error("Could not read that file."));
      reader.readAsDataURL(file);
    });
    if (request !== state.iconRequest || !$("#service-dialog").open) return;
    let icon = data.replace(/^data:image\/vnd.microsoft.icon;/, "data:image/x-icon;");
    if (/\.ico$/i.test(file.name)) icon = icon.replace(/^data:[^;]*;/, "data:image/x-icon;");
    if (!icon.startsWith("data:image/")) throw new Error("Choose a PNG, JPEG, GIF, WebP, ICO, or simple SVG image.");
    state.icon = icon;
    state.iconManual = true;
    updatePreview();
    $("#icon-status").textContent = `${file.name} · ${(file.size / 1024).toFixed(0)} KB`;
  } catch (error) { $("#icon-status").textContent = error.message; }
  event.target.value = "";
});

function setSaving(saving) {
  state.saving = saving;
  for (const control of $("#service-form").querySelectorAll("input, button")) control.disabled = saving;
}

$("#service-form").addEventListener("submit", async (event) => {
  event.preventDefault();
  if (state.saving) return;
  showError("#form-error", "");
  let url;
  try { url = normalizeURL($("#service-url").value); }
  catch (error) { showError("#form-error", error.message); $("#service-url").focus(); return; }
  const name = $("#service-name").value.trim();
  if (!name) { showError("#form-error", "Enter a service name."); $("#service-name").focus(); return; }
  const item = { name, url, description: $("#service-description").value.trim(), icon: state.icon, hidden: !$("#service-on-home").checked };
  const editing = state.editing;
  cancelIconRequest();
  setSaving(true);
  $("#save-button").textContent = "Saving…";
  try {
    const saved = await api(editing ? `/api/services/${editing.id}` : "/api/services", { method: editing ? "PUT" : "POST", body: JSON.stringify(item) });
    if (editing) state.services = state.services.map((service) => service.id === editing.id ? saved : service);
    else { state.services.push(saved); $("#search").value = ""; }
    $("#service-dialog").close();
    render();
    toast(editing ? "Changes saved." : `${saved.name} added to your services.`);
  } catch (error) { showError("#form-error", error.message); }
  finally { setSaving(false); $("#save-button").textContent = editing ? "Save changes" : "Add service"; }
});

$("#delete-button").addEventListener("click", () => {
  $("#delete-description").textContent = `“${state.editing.name}” will be removed from Harbor. The service itself will not be affected.`;
  showError("#delete-error", "");
  $("#delete-dialog").showModal();
});
$("#delete-form").addEventListener("submit", async (event) => {
  event.preventDefault();
  if (state.saving) return;
  state.saving = true;
  $("#confirm-delete").disabled = true;
  try {
    await api(`/api/services/${state.editing.id}`, { method: "DELETE" });
    state.services = state.services.filter((item) => item.id !== state.editing.id);
    $("#delete-dialog").close();
    $("#service-dialog").close();
    render();
    toast("Service removed.");
  } catch (error) { showError("#delete-error", error.message); }
  finally { state.saving = false; $("#confirm-delete").disabled = false; }
});

$("#access-button").addEventListener("click", async () => {
  if (!state.canEdit) { state.afterLogin = openSettings; openLogin(); return; }
  try {
    await api("/api/logout", { method: "POST" });
    state.canEdit = false;
    updateAccess();
    render();
    toast("Editing is locked.");
  } catch (error) { toast(error.message); }
});
$("#login-form").addEventListener("submit", async (event) => {
  event.preventDefault();
  showError("#login-error", "");
  $("#login-submit").disabled = true;
  try {
    await api("/api/login", { method: "POST", body: JSON.stringify({ password: $("#password").value }) });
    state.canEdit = true;
    $("#login-dialog").close();
    $("#password").value = "";
    updateAccess();
    render();
    toast("Editing unlocked.");
    const next = state.afterLogin;
    state.afterLogin = null;
    if (next) next();
  } catch (error) { showError("#login-error", error.message); }
  finally { $("#login-submit").disabled = false; }
});

document.querySelectorAll("[data-close]").forEach((button) => button.addEventListener("click", () => { if (!state.saving) document.getElementById(button.dataset.close).close(); }));
document.querySelectorAll("dialog").forEach((dialog) => {
  dialog.addEventListener("cancel", (event) => { if (state.saving) event.preventDefault(); });
  dialog.addEventListener("click", (event) => {
    if (event.target !== dialog || state.saving) return;
    const rect = dialog.getBoundingClientRect();
    if (event.clientX < rect.left || event.clientX > rect.right || event.clientY < rect.top || event.clientY > rect.bottom) dialog.close();
  });
});
$("#service-dialog").addEventListener("close", () => {
  cancelIconRequest();
  const destination = state.editorReturn;
  state.editorReturn = null;
  if (destination === "settings") openSettings();
  else if (destination === "apps") openLibrary();
});
$("#login-dialog").addEventListener("close", () => { $("#password").value = ""; });
document.addEventListener("contextmenu", (event) => {
  if (event.shiftKey || event.target.closest("input, textarea, select, [contenteditable], dialog")) return;
  event.preventDefault();
  openLibrary();
});
document.addEventListener("keydown", (event) => {
  if (event.key === "/" && !event.ctrlKey && !event.metaKey && !event.altKey && !document.querySelector("dialog[open]") && !["INPUT", "TEXTAREA"].includes(document.activeElement.tagName)) {
    event.preventDefault();
    openLibrary(true);
  }
});

load();
