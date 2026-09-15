# Data Model

The data directory contains one durable file, `services.json`. Its current
schema version is `1`, independent of the application release in `VERSION`.

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
