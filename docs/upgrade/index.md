---
title: Upgrading to v2
layout: default
nav_order: 5
---

# Upgrading from 1.x to v2

v2 rebuilds mysql-bkup on a shared streaming core: dump, compression, encryption and upload run in one pipe, without temporary files, whatever the size of the database. Existing commands, flags and environment variables keep working.


Everything that worked in 1.x keeps working; a few behaviours changed on purpose.

- **Image:** Debian 13 slim instead of Alpine, with Oracle's `mysqldump` for MySQL servers (correct MySQL 8 dumps) and `mariadb-dump` for MariaDB. `/bin/sh`, `/backup`, `/config`, the `backup`/`restore`/`migrate` shortcuts and the `bkup` alias are still there; `tini` is now PID 1.
- **Renamed variables** are still read, with a deprecation warning: `S3_ENDPOINT` → `AWS_S3_ENDPOINT`, `BUCKET_NAME`/`AWS_BUCKET_NAME` → `AWS_S3_BUCKET_NAME`, `ACCESS_KEY`/`SECRET_KEY` → `AWS_ACCESS_KEY`/`AWS_SECRET_KEY`, `AWS_S3_PATH`/`S3_PATH` → `REMOTE_PATH`, `FTP_HOST_NAME` → `FTP_HOST`, `BACKUP_TMP_DIR` → `TMP_DIR`.
- **`DB_USER` is the database user variable** (also `TARGET_DB_USER`). `DB_USERNAME` and `TARGET_DB_USERNAME` are still read without a warning, including the per-database forms (`DB_USERNAME_<NAME>`, `<NAME>_DB_USERNAME`).
- **`DB_SSL_MODE`** takes MySQL's mode names; `0`/`1` still work.
- **`STORAGE` and `STORAGE_PATH` env are honoured** (1.x always used the `--storage` default and `/backup`). Unknown storage names are an error instead of a silent local backup.
- **SFTP variables are renamed, and the old names are no longer read:** `SSH_HOST`/`SSH_HOST_NAME` → `SFTP_HOST`, `SSH_PORT` → `SFTP_PORT`, `SSH_USER` → `SFTP_USER`, `SSH_PASSWORD` → `SFTP_PASSWORD`, `SSH_IDENTIFY_FILE`/`SSH_IDENTITY_FILE` → `SFTP_IDENTITY_FILE`, `SSH_REMOTE_PATH` → `REMOTE_PATH`.
- **SFTP host keys are verified by default.** Set `SFTP_KNOWN_HOSTS=/path/known_hosts`, or `SFTP_INSECURE_IGNORE_HOST_KEY=true` to keep the 1.x behaviour.
- **`MAIL_SKIP_TLS` is read with its documented meaning** (`true` skips certificate verification). 1.x had it inverted, so `MAIL_SKIP_TLS=false` (the value in `.env.example`) turned verification *off*. Use `MAIL_INSECURE_SKIP_VERIFY` from now on.
- **Retention only deletes indexed backups** (see the [configuration reference](../reference/)). 1.x deleted every file older than N days in the folder. Set `BACKUP_RETENTION_ADOPT_LEGACY=true` once to let retention handle 1.x backups, or `BACKUP_INDEX=false` for name-based retention.
- **Encryption fails closed**: a `GPG_PUBLIC_KEY` that cannot be read is an error (1.x uploaded plaintext). The env var now wins over the auto-detected `/config/gnupg` key.
- **Errors no longer kill the daemon**: a failed run is logged, notified and retried at the next schedule; other databases of the same run continue (`BACKUP_RESCUE_MODE` is no longer needed).
- **Restore** works from every storage (1.x had no SFTP restore), is case-insensitive about the storage name, and no longer goes through `sh -c`.
- **`migrate`** streams directly into the target (no intermediate file), creates the target database if needed, and `TARGET_DB_NAME` defaults to the source name; `-a` migrates every database.
- **Notification templates** are built in; custom ones still go in `/config/templates` (`TEMPLATES_DIR`).
- **Version** is injected with `-ldflags "-X main.Version=..."` (was `utils.Version`).


## Checklist

1. **SFTP users:** mount a `known_hosts` file and set `SFTP_KNOWN_HOSTS`, or set `SFTP_INSECURE_IGNORE_HOST_KEY=true`.
2. **Email notifications with `MAIL_SKIP_TLS=false`:** certificate verification is now on, as the name always promised. Remove the variable, or set `MAIL_INSECURE_SKIP_VERIFY=true` if your SMTP server has a self-signed certificate.
3. **Retention:** set `BACKUP_RETENTION_ADOPT_LEGACY=true` for one run if retention should also delete the backups made by 1.x.
4. **Quoted values in Compose files:** in the list syntax (`- AWS_DISABLE_SSL="false"`) the quotes are part of the value. v2 validates booleans and rejects `"false"`; write `- AWS_DISABLE_SSL=false`.
5. **`--user` runs:** `/backup` and `/tmp/backup` are writable by any user (sticky, like `/tmp`); mounted volumes need the usual permissions.
6. **Restoring 1.x backups:** nothing to do. `.sql`, `.sql.gz`, `.sql.gpg` and `.sql.gz.gpg` files from 1.x restore as before.

## New in v2

- MySQL servers are dumped with Oracle `mysqldump` 8.4, MariaDB servers with `mariadb-dump` 11.8 (flavour detected automatically).
- `DB_URL` (`mysql://user:password@host:3306/db?ssl-mode=required`), TLS modes, client certificates.
- age encryption, zstd compression, Google Cloud Storage, FTPS, extra storage copies.
- `verify`, `list`, `delete`, `prune`, `reconcile`, `config`, `health` commands; backup index with sha256 per backup.
- Slack, Discord, ntfy, webhook and healthcheck notifications; Prometheus metrics in daemon mode.
- `migrate -a`, and migrate without an intermediate file.
