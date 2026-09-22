# Data Model

The data directory contains `services.json` and, once the page is customized,
`site.json` plus at most one `background-*` image. Each JSON schema version is
currently `1`, independent of the application release in `VERSION`.

```json
{
  "version": 1,
  "services": [
    {
      "id": "0123456789abcdef01234567",
      "name": "Media",
      "url": "http://media.example:8096",
      "description": "Movies and TV",
      "icon": "",
      "hidden": false
    }
  ]
}
```

| Field | Contract |
| --- | --- |
| `id` | Server-generated 12 random bytes encoded as hex; unique and nonempty in saved data |
| `name` | Trimmed, 1–80 Unicode characters |
| `url` | HTTP(S), at most 2,048 bytes, no embedded credentials |
| `description` | Trimmed, at most 160 Unicode characters |
| `icon` | Empty for initials, otherwise a validated base64 image data URI; decoded limit 512 KiB |
| `hidden` | `true` excludes the service from home, not All apps or the public API |

The collection has a limit of 500 services. Array order follows insertion order;
editing preserves position. There is no sorting or grouping field.

## Compatibility and Durability

An absent file starts an empty collection. The first successful mutation creates
the file. Invalid JSON, unsupported schema versions, duplicate/missing IDs, and
invalid saved services fail startup without silently replacing the file.

Older records without `hidden` deserialize to `false`, preserving existing home
cards. Retain this behavior when changing the data model.

Each mutation holds the store lock, creates a temporary `.services-*` file in the
same directory, writes and syncs the complete snapshot, closes it, and renames it
over the saved file. Memory changes only after the rename succeeds.

This is a single-process store, with no cross-process lock or shared-storage
coordination. It is not a backup system or a guarantee against every filesystem
or power-loss failure. Keep private backups as described in
[data and reliability](../operations/data.md).

## Site Settings

`site.json` holds the page title, page icon, and custom background shared by
every device. An absent file, empty title, or empty icon uses Harbor's defaults.

```json
{
  "version": 1,
  "title": "Home",
  "icon": "",
  "background": "background-0123456789abcdef01234567.jpg"
}
```

| Field | Contract |
| --- | --- |
| `title` | Trimmed, at most 60 Unicode characters, no control characters; empty means `Harbor` |
| `icon` | Empty for the built-in icon, otherwise the same validated data URI format as a service icon |
| `background` | Empty, or a store-generated `background-<24 hex>.<jpg\|png\|gif\|webp>` file in the data directory |

Backgrounds are validated by content and limited to 10 MiB and 16,384 pixels
per side (WebP is checked by signature only). An upload writes a new file,
switches `site.json` to it, and then removes the previous image, so a failure
keeps the previous background. Invalid `site.json` fails startup without
replacing the file; a referenced background that is missing is logged and
treated as absent. Which background a device displays is a browser preference.
