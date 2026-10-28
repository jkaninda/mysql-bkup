# MYSQL-BKUP

**MYSQL-BKUP** backs up, restores and migrates **MySQL and MariaDB** databases. It runs as a Docker image (one-shot, as a scheduled daemon, or as a Kubernetes Job/CronJob) and streams every backup end to end: dump, compression, encryption and upload happen in one pipe, without temporary dump files and with a fixed memory footprint, whatever the database size.

[![Tests](https://github.com/jkaninda/mysql-bkup/actions/workflows/tests.yml/badge.svg)](https://github.com/jkaninda/mysql-bkup/actions/workflows/tests.yml)
[![Build](https://github.com/jkaninda/mysql-bkup/actions/workflows/release.yml/badge.svg)](https://github.com/jkaninda/mysql-bkup/actions/workflows/release.yml)
[![Go Report](https://goreportcard.com/badge/github.com/jkaninda/mysql-bkup)](https://goreportcard.com/report/github.com/jkaninda/mysql-bkup)
![Docker Image Size (latest by date)](https://img.shields.io/docker/image-size/jkaninda/mysql-bkup?style=flat-square)
![Docker Pulls](https://img.shields.io/docker/pulls/jkaninda/mysql-bkup?style=flat-square)
<a href="https://ko-fi.com/jkaninda"><img src="https://uploads-ssl.webflow.com/5c14e387dab576fe667689cf/5cbed8a4ae2b88347c06c923_BuyMeACoffee_blue.png" height="20" alt="buy ma a coffee"></a>

Documentation: <https://jkaninda.github.io/mysql-bkup>

## Features

- **MySQL and MariaDB, each with its own tools.** The server flavour is detected (`SELECT VERSION()`): MariaDB is dumped with `mariadb-dump` and restored with `mariadb`, MySQL 8+ with Oracle's `mysqldump` and `mysql` (8.4 LTS client). Tested against MySQL 8.0, MySQL 8.4 and MariaDB 11.8.
- **Storage:** local, Amazon S3 and S3-compatible (MinIO, Wasabi, R2, ...), Google Cloud Storage, Azure Blob, SFTP, FTP/FTPS. Extra copies to more storages from the config file.
- **Encryption:** GPG (passphrase or public key, the default and compatible with 1.x) or [age](https://age-encryption.org) (recipients or passphrase). Encryption fails closed: a configured key that cannot be loaded is an error, never a silent plaintext upload.
- **Compression:** gzip (default, parallel) or zstd; restore detects the format.
- **Backup index and safe retention:** every storage folder has an index (`mysql-bkup-metadata.json`) and a `.meta.json` sidecar per backup with its sha256. Retention by age and/or count only deletes backups the index knows, and always keeps the newest one.
- **Commands:** `backup`, `restore`, `migrate`, `verify`, `list`, `delete`, `prune`, `reconcile`, `config`, `health`.
- **Scheduling:** cron expressions (5 or 6 fields, `@daily`, `@every 6h`), no overlapping runs, clean shutdown on SIGTERM, Prometheus metrics, Docker `HEALTHCHECK`.
- **Notifications:** email, Telegram, Slack, Discord, ntfy, generic webhooks (HMAC-signed), healthchecks.io-style pings.
- **Security:** no shell anywhere; database passwords go to the client tools through a 0600 option file in a private directory (never on the command line), and every secret can be read from `<NAME>_FILE` (Docker/Kubernetes secrets).
- **Multi-arch image:** `amd64`, `arm64`, `arm/v7` (see [Client tools](#client-tools)).

## Quick start

Back up the database `shop` into `./backup`:

```shell
docker run --rm --network your_network \
  -v $PWD/backup:/backup \
  -e DB_HOST=mysql -e DB_USER=root -e DB_PASSWORD=password \
  jkaninda/mysql-bkup backup -d shop
```

The last line of the output names the backup:

```
level=INFO msg="Backup completed" database=shop file=shop_20261027_031500.sql.gz size="1.2 MiB" duration=2.1s storage=local location=/backup
```

Restore it:

```shell
docker run --rm --network your_network \
  -v $PWD/backup:/backup \
  -e DB_HOST=mysql -e DB_USER=root -e DB_PASSWORD=password \
  jkaninda/mysql-bkup restore -d shop -f shop_20261027_031500.sql.gz
```

Restore accepts mysql-bkup backups and any plain `.sql` or gzip `.sql.gz` dump (encrypted or not). A missing database is created first.

### Docker Compose, scheduled

```yaml
services:
  mysql-bkup:
    image: jkaninda/mysql-bkup
    command: backup -d shop --cron-expression "@daily"
    volumes:
      - ./backup:/backup
    environment:
      - DB_HOST=mysql
      - DB_USER=backup
      - DB_PASSWORD_FILE=/run/secrets/db_password
      - BACKUP_RETENTION_DAYS=7
      - GPG_PASSPHRASE=change-me
```

### Kubernetes CronJob

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: mysql-bkup
spec:
  schedule: "0 1 * * *"
  jobTemplate:
    spec:
      template:
        spec:
          restartPolicy: Never
          containers:
            - name: mysql-bkup
              image: jkaninda/mysql-bkup
              args: ["backup", "-d", "shop", "--storage", "s3", "--path", "/mysql"]
              envFrom:
                - secretRef:
                    name: mysql-bkup # DB_HOST, DB_USER, DB_PASSWORD, AWS_* ...
```

## Commands

| Command | Purpose |
|---|---|
| `backup` | Back up now, or as a daemon with `--cron-expression` / `BACKUP_CRON_EXPRESSION`. `-d DB`, `-a` (every database, one file each), `-A` (every database in one file), `--exclude-db`, `--custom-name`, `--disable-compression`, `-c config.yaml` |
| `restore` | Stream a backup back: `-f FILE [-d DB]`. Without `-d`, the dump's own `USE` statements decide (all-in-one backups) |
| `migrate` | Copy `DB_NAME` (or every database with `-a`) straight into `TARGET_DB_*`: `dump \| restore` through a pipe, no file |
| `verify` | Download, decrypt and decompress a backup and check its sha256, without restoring |
| `list` | Backup history from the index (`--json`, `--storage-scan`) |
| `delete` | Delete a backup, its sidecar and its index entry |
| `prune` | Apply the retention policy (`--dry-run`) |
| `reconcile` | Rebuild the index from the sidecars (`--from-index OLD` after renaming it) |
| `config` | Show the resolved configuration with sources, secrets redacted; `--env-docs` prints the reference below |
| `health` | Docker `HEALTHCHECK` for the daemon |
| `version` | Print the version |

The image also has the `backup`, `restore` and `migrate` shortcuts (`docker run ... --entrypoint backup`, or in Kubernetes `command: [backup]`) and the `bkup` alias.

### Exit codes

`0` success, `1` failure, `2` configuration error, `3` database connection or authentication, `4` storage, `5` encryption/decryption.

## Storage

```shell
# S3 / MinIO
-e AWS_S3_ENDPOINT=https://s3.amazonaws.com -e AWS_S3_BUCKET_NAME=backups -e AWS_REGION=us-east-1 \
-e AWS_ACCESS_KEY=... -e AWS_SECRET_KEY=... \
jkaninda/mysql-bkup backup -d shop --storage s3 --path /mysql
# MinIO and other S3 alternatives: add AWS_FORCE_PATH_STYLE=true (and AWS_DISABLE_SSL=true for plain http).
# Without AWS_ACCESS_KEY, the AWS default credential chain is used (IAM role, IRSA, profile).

# SFTP (host keys are verified: mount a known_hosts file)
-e SFTP_HOST=backup.example.com -e SFTP_USER=backup -e SFTP_IDENTITY_FILE=/config/id_ed25519 \
-e SFTP_KNOWN_HOSTS=/config/known_hosts \
jkaninda/mysql-bkup backup -d shop --storage sftp --path /home/backup/mysql

# FTP / FTPS
-e FTP_HOST=ftp.example.com -e FTP_USER=backup -e FTP_PASSWORD=... -e FTP_TLS=true \
jkaninda/mysql-bkup backup -d shop --storage ftp --path /mysql

# Azure Blob
-e AZURE_STORAGE_ACCOUNT_NAME=... -e AZURE_STORAGE_ACCOUNT_KEY=... -e AZURE_STORAGE_CONTAINER_NAME=backups \
jkaninda/mysql-bkup backup -d shop --storage azure --path mysql

# Google Cloud Storage
-e GCS_BUCKET=backups -e GCS_CREDENTIALS_FILE=/config/gcs.json \
jkaninda/mysql-bkup backup -d shop --storage gcs --path mysql
```

`STORAGE` (env) and `--storage` are equivalent; `ssh` and `remote` are accepted for `sftp`. The object key is exactly `<path>/<file>`.

## Encryption

| Mode | Backup | Restore | File |
|---|---|---|---|
| GPG passphrase | `GPG_PASSPHRASE` | `GPG_PASSPHRASE` | `.sql.gz.gpg` |
| GPG public key | `GPG_PUBLIC_KEY` (path or inline), or `/config/gnupg/public_key.asc` | `GPG_PRIVATE_KEY` (+ `GPG_PRIVATE_KEY_PASSPHRASE`) | `.sql.gz.gpg` |
| age recipients | `AGE_RECIPIENTS` (`age1...`, `ssh-ed25519 ...`) or `AGE_RECIPIENTS_FILE` | `AGE_IDENTITY` / `AGE_IDENTITY_FILE` (default `/config/age/identity.txt`) | `.sql.gz.age` |
| age passphrase | `AGE_PASSPHRASE` | `AGE_PASSPHRASE` | `.sql.gz.age` |

`ENCRYPTION=auto` (default) picks the mode from the keys that are set; GPG wins when both GPG and age keys are set. Restore detects the format from the content.

## Backup index and retention

Each storage folder holds an index file (`BACKUP_INDEX_NAME`, default `{tool}-metadata.json`; placeholders `{tool}`, `{engine}` (always `mysql`), `{name}`, `{db}`) and one `<backup>.meta.json` sidecar per backup. `list`, `delete`, `prune` and retention work from the index, so mysql-bkup only ever deletes files it created itself, and several instances can share a folder with different index names.

- `BACKUP_RETENTION_DAYS=N` deletes backups older than N days, `BACKUP_RETENTION_COUNT=N` keeps the newest N. The newest successful backup is always kept.
- Backups made before the index existed (1.x) are never deleted, unless `BACKUP_RETENTION_ADOPT_LEGACY=true` imports them.
- `BACKUP_INDEX=false` turns the index off; retention then only matches this tool's file names directly in the folder.

## Multiple databases

```shell
backup -a                      # every database, one file each (system databases are skipped)
backup -a --exclude-db test,tmp
backup -A                      # everything in one file: all_databases_<timestamp>.sql.gz
```

Or a YAML file (`-c /config/config.yaml`, also picked up automatically at `/config/config.yaml`); the 1.x format still works:

```yaml
cronExpression: "@daily"
databases:
  - name: shop
    host: mysql1
    user: backup
    password: ${SHOP_PASSWORD}
    path: /shop
  - name: blog          # credentials from DB_USER_BLOG / BLOG_DB_USER ... or the global DB_*
```

## Client tools

| Server | Dump | Restore |
|---|---|---|
| MariaDB | `mariadb-dump` (MariaDB 11.8 LTS client) | `mariadb` |
| MySQL 8.0 / 8.4 / 9.x | Oracle `mysqldump` 8.4 LTS | Oracle `mysql` 8.4 LTS |

Dumps use `--single-transaction --routines --triggers --events --hex-blob --quick`, plus `--no-tablespaces --set-gtid-purged=OFF` on MySQL. A single-database dump has no `CREATE DATABASE`/`USE`, so it can be restored under another name.

Restores into a named database (`-d`) also accept dumps written elsewhere: the `CREATE DATABASE`, `DROP DATABASE` and `USE` statements of a `mysqldump --databases` dump are skipped, so its objects land in the target database whatever the source was called (a dump of several databases is refused). `DEFINER` clauses are removed when the restoring user may not name another definer (`RESTORE_NO_DEFINER=auto`): views, triggers, routines and events then belong to that user. Dumps from MariaDB and from older MySQL versions restore into MySQL 8: the MariaDB sandbox line is dropped for the Oracle client, MariaDB's `utf8mb4_uca1400_ai_ci` (`_as_ci`, `_as_cs`) collations become `utf8mb4_0900_*`, and `sql_mode` values keep only the modes MySQL 8 knows (`NO_AUTO_CREATE_USER` goes). Data is never rewritten. On MySQL with binary logging on (the default), creating functions and triggers without `SUPER` also needs the server setting `log_bin_trust_function_creators=1`.

The image is based on Debian 13 (trixie) slim. Oracle publishes its client for `amd64` and `arm64` only: on `arm/v7` the image has the MariaDB tools only, and MySQL servers are dumped with `mariadb-dump` (a warning is logged). `MYSQL_CLIENT=mysql|mariadb` forces one family.

TLS: `DB_SSL_MODE=disabled` (default, as in 1.x) `| preferred | required | verify_ca | verify_identity`, with `DB_SSL_CA` (used for the verify modes), `DB_SSL_CERT`, `DB_SSL_KEY`. 1.x values `0` and `1` still mean disabled and required. With the MariaDB client, `verify_ca` also checks the host name.

Restores are not atomic: MySQL cannot roll back DDL, so a restore that fails halfway leaves the statements before the failure applied (the exit code is non-zero). Restore into a new database, or keep a backup of the target, when that matters.

## Use with Miabi

Miabi runs `backup -d <db>`, `backup --storage s3 -d <db> --path <path>` and `restore -d <db> -f <file> [--storage s3 --path <path>]` with `DB_*`, `AWS_*` and `GPG_PASSPHRASE`, and reads the artifact name from the last `file=` in the output. That contract is unchanged and covered by the integration tests.

## Upgrading from 1.x

Everything that worked in 1.x keeps working; a few behaviours changed on purpose.

- **Image:** Debian 13 slim instead of Alpine, with Oracle's `mysqldump` for MySQL servers (correct MySQL 8 dumps) and `mariadb-dump` for MariaDB. `/bin/sh`, `/backup`, `/config`, the `backup`/`restore`/`migrate` shortcuts and the `bkup` alias are still there; `tini` is now PID 1.
- **Renamed variables** are still read, with a deprecation warning: `DB_USER` → `DB_USER`, `S3_ENDPOINT` → `AWS_S3_ENDPOINT`, `BUCKET_NAME`/`AWS_BUCKET_NAME` → `AWS_S3_BUCKET_NAME`, `ACCESS_KEY`/`SECRET_KEY` → `AWS_ACCESS_KEY`/`AWS_SECRET_KEY`, `AWS_S3_PATH`/`S3_PATH`/`REMOTE_PATH` → `REMOTE_PATH`, `SFTP_HOST` → `SFTP_HOST`, `SFTP_IDENTITY_FILE` → `SFTP_IDENTITY_FILE`, `FTP_HOST_NAME` → `FTP_HOST`, `BACKUP_TMP_DIR` → `TMP_DIR`.
- **`DB_SSL_MODE`** takes MySQL's mode names; `0`/`1` still work.
- **`STORAGE` and `STORAGE_PATH` env are honoured** (1.x always used the `--storage` default and `/backup`). Unknown storage names are an error instead of a silent local backup.
- **SFTP variables are renamed, and the old names are no longer read:** `SSH_HOST`/`SSH_HOST_NAME` → `SFTP_HOST`, `SSH_PORT` → `SFTP_PORT`, `SSH_USER` → `SFTP_USER`, `SSH_PASSWORD` → `SFTP_PASSWORD`, `SSH_IDENTIFY_FILE`/`SSH_IDENTITY_FILE` → `SFTP_IDENTITY_FILE`, `SSH_REMOTE_PATH` → `REMOTE_PATH`.
- **SFTP host keys are verified by default.** Set `SFTP_KNOWN_HOSTS=/path/known_hosts`, or `SFTP_INSECURE_IGNORE_HOST_KEY=true` to keep the 1.x behaviour.
- **`MAIL_SKIP_TLS` is read with its documented meaning** (`true` skips certificate verification). 1.x had it inverted, so `MAIL_SKIP_TLS=false` (the value in `.env.example`) turned verification *off*. Use `MAIL_INSECURE_SKIP_VERIFY` from now on.
- **Retention only deletes indexed backups** (see above). 1.x deleted every file older than N days in the folder. Set `BACKUP_RETENTION_ADOPT_LEGACY=true` once to let retention handle 1.x backups, or `BACKUP_INDEX=false` for name-based retention.
- **Encryption fails closed**: a `GPG_PUBLIC_KEY` that cannot be read is an error (1.x uploaded plaintext). The env var now wins over the auto-detected `/config/gnupg` key.
- **Errors no longer kill the daemon**: a failed run is logged, notified and retried at the next schedule; other databases of the same run continue (`BACKUP_RESCUE_MODE` is no longer needed).
- **Restore** works from every storage (1.x had no SFTP restore), is case-insensitive about the storage name, and no longer goes through `sh -c`.
- **`migrate`** streams directly into the target (no intermediate file), creates the target database if needed, and `TARGET_DB_NAME` defaults to the source name; `-a` migrates every database.
- **Notification templates** are built in; custom ones still go in `/config/templates` (`TEMPLATES_DIR`).
- **Version** is injected with `-ldflags "-X main.Version=..."` (was `utils.Version`).

## Building

```shell
make build            # bin/mysql-bkup
make test             # unit tests
make test-integration # Docker: MySQL 8.4/8.0, MariaDB 11.8, MinIO
make docker-build VERSION=v2.0.0
make env-docs         # regenerate the reference below
```

## Environment variables

Generated with `mysql-bkup config --env-docs`. Every secret also accepts `<NAME>_FILE`. Resolution order: flag, then env, then deprecated names, then the YAML `env:` section, then the default.

### storage

| Variable | Flag | Default | Fallbacks | Description |
|---|---|---|---|---|
| `STORAGE` | `--storage` `-s` | `local` |  | Storage backend: local, s3, gcs, azure, sftp (ssh, remote), ftp (local, s3, gcs, azure, sftp, ssh, ftp, remote) |
| `STORAGE_PATH` |  | `/backup` |  | Root directory of the local storage backend |
| `REMOTE_PATH` | `--path` `-P` |  | `AWS_S3_PATH` (deprecated), `S3_PATH` (deprecated) | Folder inside the storage backend (without file name) |

### general

| Variable | Flag | Default | Fallbacks | Description |
|---|---|---|---|---|
| `BACKUP_NAME` | `--name` `-n` |  |  | Base name of the artifact; a timestamp is appended (default: the database or volume name) |
| `BACKUP_CUSTOM_NAME` | `--custom-name` |  |  | Fixed artifact name without timestamp (overwrites the previous one) |
| `FILE_NAME` | `--file` `-f` |  |  | Backup file to restore, verify or delete |
| `BACKUP_CONFIG_FILE` | `--config` `-c` |  |  | YAML configuration file (default /config/config.yaml if present) |
| `BACKUP_TIMEOUT` |  |  |  | Abort a backup after this duration (0 = none) |
| `RESTORE_TIMEOUT` |  |  |  | Abort a restore after this duration (0 = none) |
| `TMP_DIR` |  | `/tmp/backup` | `BACKUP_TMP_DIR` (deprecated) | Scratch directory (only for secret files and disk-based modes) |
| `COMPRESSION` | `--compression` | `gzip` |  | Compression: gzip, zstd or none (gzip, zstd, none) |
| `COMPRESSION_LEVEL` |  |  |  | Compression level (0 = default) |
| `DISABLE_COMPRESSION` | `--disable-compression` |  |  | Same as COMPRESSION=none |
| `RESTORE_VERIFY_FIRST` | `--verify-first` | `auto` |  | Verify the whole artifact before restoring (auto: on for GPG-encrypted volumes) (auto, true, false) |
| `LOG_LEVEL` |  | `info` |  | Log level (debug, info, warn, error) |
| `LOG_FORMAT` |  | `text` |  | Log format (text, json) |
| `TIME_FORMAT` |  | `2006-01-02 15:04:05` |  | Time layout used in notifications |
| `RESULT_FILE` | `--result-file` |  |  | Write the run result as JSON to this path |

### scheduling

| Variable | Flag | Default | Fallbacks | Description |
|---|---|---|---|---|
| `BACKUP_CRON_EXPRESSION` | `--cron-expression` `-e` |  |  | Run as a daemon on this cron schedule (5 or 6 fields, @daily, @every 6h) |
| `BACKUP_RUN_ON_START` |  |  |  | In daemon mode, run one backup immediately at start |
| `BACKUP_RESCUE_MODE` |  |  |  | Deprecated: runs always continue with the remaining targets after a failure |
| `METRICS_ADDR` |  |  |  | Serve Prometheus /metrics and /healthz on this address in daemon mode (e.g. :9090) |
| `HEALTH_FILE` |  | `/tmp/bkup-health.json` |  | State file read by the health command |
| `HEALTH_MAX_AGE` |  |  |  | health fails when the last success is older than this (default: 2x the schedule interval) |

### retention

| Variable | Flag | Default | Fallbacks | Description |
|---|---|---|---|---|
| `BACKUP_RETENTION_DAYS` |  | `0` |  | Delete backups older than N days (0 = keep) |
| `BACKUP_RETENTION_COUNT` |  | `0` |  | Keep only the newest N backups (0 = no limit) |
| `BACKUP_RETENTION_ALLOW_ROOT` |  |  |  | Allow pattern-based retention at the storage root (index disabled) |
| `BACKUP_RETENTION_ADOPT_LEGACY` |  |  |  | Import pre-index backups matching this tool's naming into the index |
| `BACKUP_INDEX` |  | `true` |  | Keep the backup index file and per-backup .meta.json sidecars |
| `BACKUP_INDEX_NAME` | `--index-name` | `{tool}-metadata.json` |  | Index file name; placeholders {tool} {engine} {name} {db} |

### storage-s3

| Variable | Flag | Default | Fallbacks | Description |
|---|---|---|---|---|
| `AWS_S3_ENDPOINT` |  |  | `S3_ENDPOINT` (deprecated), `AWS_ENDPOINT_URL_S3`, `AWS_ENDPOINT_URL` | S3 endpoint (empty = AWS) |
| `AWS_S3_BUCKET_NAME` |  |  | `BUCKET_NAME` (deprecated), `AWS_BUCKET_NAME` (deprecated) | S3 bucket |
| `AWS_REGION` |  | `us-east-1` | `AWS_DEFAULT_REGION` | S3 region |
| `AWS_ACCESS_KEY` |  |  | `ACCESS_KEY` (deprecated), `AWS_ACCESS_KEY_ID` | S3 access key (empty = default AWS credential chain) |
| `AWS_SECRET_KEY` |  |  | `SECRET_KEY` (deprecated), `AWS_SECRET_ACCESS_KEY`, `AWS_SECRET_KEY_FILE` | S3 secret key |
| `AWS_SESSION_TOKEN` |  |  | `AWS_SESSION_TOKEN_FILE` | S3 session token |
| `AWS_PROFILE` |  |  |  | AWS shared config profile |
| `AWS_DISABLE_SSL` |  |  |  | Use http for the S3 endpoint |
| `AWS_FORCE_PATH_STYLE` |  |  |  | Path-style S3 addressing (MinIO) |
| `S3_STORAGE_CLASS` |  |  |  | S3 storage class |
| `S3_SSE` |  |  |  | Server-side encryption: AES256 or aws:kms (aes256, aws:kms) |
| `S3_SSE_KMS_KEY_ID` |  |  |  | KMS key id for aws:kms |
| `S3_PART_SIZE` |  | `64MiB` |  | Multipart part size; max object size is 10000 x this |
| `S3_UPLOAD_CONCURRENCY` |  | `2` |  | Parallel part uploads (memory = part size x this) |

### storage-sftp

| Variable | Flag | Default | Fallbacks | Description |
|---|---|---|---|---|
| `SFTP_HOST` |  |  |  | SFTP host |
| `SFTP_PORT` |  | `22` |  | SFTP port |
| `SFTP_USER` |  |  |  | SFTP user |
| `SFTP_PASSWORD` |  |  | `SFTP_PASSWORD_FILE` | SFTP password |
| `SFTP_IDENTITY_FILE` |  |  |  | SFTP private key file (the SSH agent is also used when SSH_AUTH_SOCK is set) |
| `SFTP_IDENTITY_PASSPHRASE` |  |  | `SFTP_IDENTITY_PASSPHRASE_FILE` | Passphrase of the SFTP private key |
| `SFTP_KNOWN_HOSTS` |  |  |  | known_hosts file used to verify the SFTP host key (default ~/.ssh/known_hosts) |
| `SFTP_INSECURE_IGNORE_HOST_KEY` |  |  |  | Do not verify the SFTP host key (insecure) |

### storage-ftp

| Variable | Flag | Default | Fallbacks | Description |
|---|---|---|---|---|
| `FTP_HOST` |  |  | `FTP_HOST_NAME` (deprecated) | FTP host |
| `FTP_PORT` |  | `21` |  | FTP port |
| `FTP_USER` |  |  |  | FTP user |
| `FTP_PASSWORD` |  |  | `FTP_PASSWORD_FILE` | FTP password |
| `FTP_TLS` |  |  |  | Use explicit FTPS |

### storage-azure

| Variable | Flag | Default | Fallbacks | Description |
|---|---|---|---|---|
| `AZURE_STORAGE_ACCOUNT_NAME` |  |  |  | Azure storage account |
| `AZURE_STORAGE_ACCOUNT_KEY` |  |  | `AZURE_STORAGE_ACCOUNT_KEY_FILE` | Azure storage account key |
| `AZURE_STORAGE_CONTAINER_NAME` |  |  |  | Azure blob container |
| `AZURE_STORAGE_ENDPOINT` |  |  |  | Azure blob endpoint (Azurite, sovereign clouds) |
| `AZURE_STORAGE_SAS_TOKEN` |  |  | `AZURE_STORAGE_SAS_TOKEN_FILE` | Azure SAS token (instead of the account key) |

### storage-gcs

| Variable | Flag | Default | Fallbacks | Description |
|---|---|---|---|---|
| `GCS_BUCKET` |  |  |  | GCS bucket |
| `GCS_CREDENTIALS_FILE` |  |  | `GOOGLE_APPLICATION_CREDENTIALS` | GCS service account JSON file |
| `GCS_CREDENTIALS_JSON` |  |  | `GCS_CREDENTIALS_JSON_FILE` | GCS service account JSON content |
| `GCS_ENDPOINT` |  |  |  | GCS endpoint (emulators) |

### encryption

| Variable | Flag | Default | Fallbacks | Description |
|---|---|---|---|---|
| `ENCRYPTION` | `--encryption` | `auto` |  | auto (from the keys set), none, gpg or age (auto, none, gpg, age) |
| `ENCRYPTION_ARMOR` |  |  |  | ASCII-armor encrypted output |
| `GPG_PASSPHRASE` |  |  | `GPG_PASSPHRASE_FILE` | GPG symmetric passphrase |
| `GPG_PUBLIC_KEY` |  |  |  | GPG public key(s): file path or inline key (default /config/gnupg/public_key.{asc,gpg}) |
| `GPG_PRIVATE_KEY` |  |  | `GPG_PRIVATE_KEY_FILE` | GPG private key for restore: file path or inline key (default /config/gnupg/private_key.{asc,gpg}) |
| `GPG_PRIVATE_KEY_PASSPHRASE` |  |  | `GPG_PRIVATE_KEY_PASSPHRASE_FILE` | Passphrase of the GPG private key (default GPG_PASSPHRASE) |
| `AGE_RECIPIENTS` |  |  |  | age recipients (age1..., ssh-ed25519 ...), comma or newline separated |
| `AGE_RECIPIENTS_FILE` |  |  |  | File with age recipients |
| `AGE_PASSPHRASE` |  |  | `AGE_PASSPHRASE_FILE` | age passphrase (scrypt) |
| `AGE_IDENTITY` |  |  | `AGE_IDENTITY_FILE` | age identity (AGE-SECRET-KEY-...) for restore |
| `AGE_IDENTITY_FILE` |  |  |  | age identity file for restore (default /config/age/identity.txt) |

### notifications

| Variable | Flag | Default | Fallbacks | Description |
|---|---|---|---|---|
| `MAIL_HOST` |  |  |  | SMTP host |
| `MAIL_PORT` |  | `587` |  | SMTP port |
| `MAIL_USERNAME` |  |  |  | SMTP user |
| `MAIL_PASSWORD` |  |  | `MAIL_PASSWORD_FILE` | SMTP password |
| `MAIL_FROM` |  |  |  | Sender address |
| `MAIL_TO` |  |  |  | Recipients, comma separated |
| `MAIL_SKIP_TLS` |  |  |  | Deprecated: true skips certificate verification (use MAIL_INSECURE_SKIP_VERIFY) |
| `MAIL_INSECURE_SKIP_VERIFY` |  |  |  | Skip SMTP certificate verification |
| `MAIL_TLS` |  | `starttls` |  | SMTP TLS mode (starttls, tls, none) |
| `TG_TOKEN` |  |  | `TELEGRAM_BOT_TOKEN`, `TG_TOKEN_FILE` | Telegram bot token |
| `TG_CHAT_ID` |  |  | `TELEGRAM_CHAT_ID` | Telegram chat id |
| `POSTA_URL` |  |  |  | Posta base URL (https://posta.example.com) for email notifications through Posta |
| `POSTA_API_KEY` |  |  | `POSTA_API_KEY_FILE` | Posta API key with the send scope |
| `POSTA_WORKSPACE_ID` |  |  |  | Posta workspace id (only for account-wide API keys) |
| `POSTA_FROM` |  |  |  | Posta sender address (default MAIL_FROM) |
| `POSTA_TO` |  |  |  | Posta recipients, comma separated (default MAIL_TO) |
| `POSTA_TEMPLATE` |  |  |  | Posta template name for notifications; the event fields are passed as template data (default: built-in HTML) |
| `POSTA_TEMPLATE_ERROR` |  |  |  | Posta template name for failures (default POSTA_TEMPLATE) |
| `POSTA_LANGUAGE` |  |  |  | Posta template language |
| `WEBHOOK_URL` |  |  |  | Generic JSON webhook URL |
| `WEBHOOK_SECRET` |  |  | `WEBHOOK_SECRET_FILE` | HMAC-SHA256 secret for the X-Bkup-Signature header |
| `SLACK_WEBHOOK_URL` |  |  | `SLACK_WEBHOOK_URL_FILE` | Slack incoming webhook URL |
| `DISCORD_WEBHOOK_URL` |  |  | `DISCORD_WEBHOOK_URL_FILE` | Discord webhook URL |
| `NTFY_URL` |  |  |  | ntfy topic URL |
| `NTFY_TOKEN` |  |  | `NTFY_TOKEN_FILE` | ntfy access token |
| `NOTIFY_ON` |  | `always` |  | When to notify (always, failure, success, never) |
| `HEALTHCHECK_PING_URL` |  |  |  | healthchecks.io-style URL pinged on start (/start), success and failure (/fail) |
| `BACKUP_REFERENCE` |  |  |  | Free text included in notifications |
| `TEMPLATES_DIR` |  | `/config/templates` |  | Directory with custom notification templates |

### database

| Variable | Flag | Default | Fallbacks | Description |
|---|---|---|---|---|
| `DB_URL` |  |  | `DB_URL_FILE` | Connection URL mysql://user:password@host:port/database?ssl-mode=required (overrides host, port, user, password) |
| `DB_HOST` |  | `localhost` |  | Database host (a path starting with / is a Unix socket) |
| `DB_PORT` |  | `3306` |  | Database port |
| `DB_NAME` | `--dbname` `-d` |  |  | Database name |
| `DB_USER` |  |  | `DB_USERNAME` | Database user |
| `DB_PASSWORD` |  |  | `DB_PASSWORD_FILE` | Database password |
| `DB_SSL_MODE` |  | `disabled` |  | TLS mode: disabled, preferred, required, verify_ca, verify_identity (1.x values 0 and 1 mean disabled and required) |
| `DB_SSL_CA` |  | `/etc/ssl/certs/ca-certificates.crt` |  | CA certificate file, used with DB_SSL_MODE=verify_ca or verify_identity |
| `DB_SSL_CERT` |  |  |  | Client certificate file |
| `DB_SSL_KEY` |  |  |  | Client key file |
| `BACKUP_ALL_DATABASES` | `--all-databases` `-a` |  |  | Back up every database, one file each |
| `BACKUP_ALL_IN_ONE` | `--all-in-one` `-A` |  |  | Back up every database into a single file |
| `BACKUP_EXCLUDE_DATABASES` | `--exclude-db` |  |  | Databases to skip with --all-databases, comma separated |
| `TARGET_DB_URL` |  |  | `TARGET_DB_URL_FILE` | migrate: target connection URL |
| `TARGET_DB_HOST` |  |  |  | migrate: target host |
| `TARGET_DB_PORT` |  | `3306` |  | migrate: target port |
| `TARGET_DB_NAME` |  |  |  | migrate: target database (default: the source name) |
| `TARGET_DB_USER` |  |  | `TARGET_DB_USERNAME` | migrate: target user |
| `TARGET_DB_PASSWORD` |  |  | `TARGET_DB_PASSWORD_FILE` | migrate: target password |
| `TARGET_DB_SSL_MODE` |  | `disabled` |  | migrate: target TLS mode (same values as DB_SSL_MODE) |
| `TARGET_DB_SSL_CA` |  | `/etc/ssl/certs/ca-certificates.crt` |  | migrate: target CA certificate file |
| `MYSQL_CLIENT` |  | `auto` |  | Client tools: auto (mariadb-dump for MariaDB, Oracle mysqldump for MySQL), mysql or mariadb (auto, mysql, mariadb) |
| `RESTORE_NO_DEFINER` |  | `auto` |  | Remove DEFINER clauses on restore (views, triggers, routines and events then belong to the restoring user); auto = when that user lacks SET_USER_ID/SUPER (auto, true, false) |


## Available image registries

```
docker pull jkaninda/mysql-bkup
docker pull ghcr.io/jkaninda/mysql-bkup
```

Tags: `latest`, and immutable release tags (`v2.0.0`, `2.0.0`, `2.0`, `2`). Pin a release tag in production.

## PostgreSQL

Looking for PostgreSQL? See [pg-bkup](https://github.com/jkaninda/pg-bkup).

## License

This project is licensed under the MIT License. See the LICENSE file for details.


## Copyright

Copyright (c) [2023] [Jonas Kaninda]
