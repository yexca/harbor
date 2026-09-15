# Data and Reliability

`/data/services.json` holds the complete collection, including icons. The file
appears after the first successful service mutation. No database service or
separate icon directory is required.

## Back Up

Avoid edits while copying, or stop Harbor first for a stable copy:

```sh
docker compose stop harbor
docker compose cp harbor:/data/services.json ./services.backup.json
docker compose start harbor
```

The example backup filename is ignored by Git. Store real backups outside the
repository in private storage. If no service has ever been saved, there may be
no file to copy. Keep the entire file to retain icons and visibility choices.

## Restore

1. Stop Harbor and preserve a copy of the current data.
2. Replace `services.json` in the data volume or bind-mounted directory.
3. Ensure UID/GID `10001:10001` can read and write the directory and file in Docker.
4. Start Harbor and check its logs and collection.

Do not edit the file while Harbor is running. Startup rejects invalid data and
preserves it for recovery. Prefer restoring a known-good backup to reconstructing
the JSON manually. Browser appearance preferences are not part of this backup.

## Runtime Limits

The store accepts at most 500 services, with each decoded icon limited to
512 KiB. Large collections of large icons use more memory, disk, and network
bandwidth because reads and writes handle the collection as one snapshot.

Writes are synchronized inside one process and use same-directory temporary
files. This does not coordinate multiple containers sharing a volume. Do not
scale replicas against the same data directory. Keep free disk space available
for both the current file and the replacement snapshot.

The backend performs no scheduled polling. Sessions are memory-only and reset
on restart. A healthy `/healthz` does not guarantee available disk space or
that the configured NAS services can be reached.
