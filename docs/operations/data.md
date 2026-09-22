# Data and Reliability

`/data/services.json` holds the complete collection, including icons. The file
appears after the first successful service mutation. `/data/site.json` holds the
page title and icon and names the custom background image, stored beside it as
`background-*`. They appear after the first customization. No database service
or separate icon directory is required.

Both Compose configurations bind-mount `./data` beside the Compose file at
`/data`. The host file is `./data/services.json`; deleting a container or running
`docker compose down -v` does not delete this directory.

## Back Up

Avoid edits while copying, or stop Harbor first for a stable copy:

```sh
docker compose stop harbor
docker compose cp harbor:/data/services.json ./services.backup.json
docker compose start harbor
```

To keep customization too, copy the whole host `data` directory while Harbor is
stopped. `site.json` and its `background-*` image belong together.

The example backup filename is ignored by Git. Store real backups outside the
repository in private storage. If no service has ever been saved, there may be
no file to copy. Keep the entire file to retain icons and visibility choices.

## Restore

1. Stop Harbor and preserve a copy of the current data.
2. Replace `data/services.json` in the host directory, plus `site.json` and its
   `background-*` image if you backed them up.
3. Ensure UID/GID `10001:10001` can read and write the directory and file in Docker.
4. Start Harbor and check its logs and collection.

Do not edit the file while Harbor is running. Startup rejects invalid data and
preserves it for recovery. Prefer restoring a known-good backup to reconstructing
the JSON manually. Browser appearance preferences are not part of this backup.

## Migrate from a Named Volume

Changing the Compose mount does not copy existing data. Before recreating an
existing container with the new configuration, stop it and copy its saved file
into an empty host `data` directory:

```sh
docker compose stop harbor
mkdir -p data
docker compose cp harbor:/data/services.json ./data/services.json
sudo chown -R 10001:10001 data
docker compose up -d
```

These are Linux shell commands. On Windows, create the directory and copy the
file without the `chown` command. Preserve the original volume until the restored
collection has been verified. Never overwrite an existing host collection during
migration. A never-used instance may have no `services.json` to copy.

## Runtime Limits

The store accepts at most 500 services, with each decoded icon limited to
512 KiB. One custom background of up to 10 MiB is kept; replaced images are deleted. Large collections of large icons use more memory, disk, and network
bandwidth because reads and writes handle the collection as one snapshot.

Writes are synchronized inside one process and use same-directory temporary
files. This does not coordinate multiple containers sharing a directory. Do not
scale replicas against the same data directory. Keep free disk space available
for both the current file and the replacement snapshot.

The backend performs no scheduled polling. Sessions are memory-only and reset
on restart. A healthy `/healthz` does not guarantee available disk space or
that the configured NAS services can be reached.
