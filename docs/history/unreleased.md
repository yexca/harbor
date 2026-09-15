## Start page

- Add a full-screen local wallpaper, clock/date, and responsive glass service cards.
- Provide All apps through right-click, Settings, keyboard shortcut, and a bottom launcher.
- Add service editing, icon fetching/uploads, search, and shared home selection.
- Keep appearance and clock preferences in each browser.

## Runtime and data

- Serve embedded assets and a JSON API from one Go process with no third-party modules.
- Store services and icons in an atomic JSON snapshot with legacy visibility compatibility.
- Support optional editing protection and bounded icon requests.
- Package a non-root Docker runtime with persistent `/data` storage.

## Repository and development

- Initialize the main branch and establish English agent, design, and contribution guides.
- Organize user, architecture, operations, development, ADR, and history documentation.
- Add Makefile checks, staged/working-tree privacy scanning, isolated Docker smoke
  testing, and GitHub Actions validation.
- Group application code and embedded browser assets under `server/`, keeping
  the Go module and deployment/development entry points at the repository root.
