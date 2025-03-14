---
title: Configuration Reference
layout: default
nav_order: 3
---

# Configuration Reference

MySQL backup, restore, and migration processes can be configured using **environment variables** or **CLI flags**.

## CLI Utility Usage

The `mysql-bkup` CLI provides commands and options to manage MySQL backups efficiently.

| Option                  | Short Flag | Description                                                                             |
|-------------------------|------------|-----------------------------------------------------------------------------------------|
| `mysql-bkup`            | `bkup`     | CLI tool for managing MySQL backups, restoration, and migration.                        |
| `backup`                |            | Executes a backup operation.                                                            |
| `restore`               |            | Restores a database from a backup file.                                                 |
| `migrate`               |            | Migrates a database from one instance to another.                                       |
| `--storage`             | `-s`       | Specifies the storage type (`local`, `s3`, `ssh`, etc.). Default: `local`.              |
| `--file`                | `-f`       | Defines the backup file name for restoration.                                           |
| `--path`                |            | Sets the storage path (e.g., `/custom_path` for S3 or `/home/foo/backup` for SSH).      |
| `--config`              | `-c`       | Provides a configuration file for multi-database backups (e.g., `/backup/config.yaml`). |
| `--dbname`              | `-d`       | Specifies the database name to back up or restore.                                      |
| `--port`                | `-p`       | Defines the database port. Default: `3306`.                                             |
| `--disable-compression` |            | Disables compression for database backups.                                              |
| `--cron-expression`     | `-e`       | Schedules backups using a cron expression (e.g., `0 0 * * *` or `@daily`).              |
| `--all-databases`       | `-a`       | Backs up all databases separately (e.g., `backup --all-databases`).                     |
| `--all-in-one`          | `-A`       | Backs up all databases in a single file (e.g., `backup --all-databases --single-file`). |
| `--custom-name`         | ``         | Sets custom backup name for one time backup                                             |
| `--help`                | `-h`       | Displays the help message and exits.                                                    |
| `--version`             | `-V`       | Shows version information and exits.                                                    |

---

## Environment Variables

| Name                           | Requirement                          | Description                                                                |
|--------------------------------|--------------------------------------|----------------------------------------------------------------------------|
| `DB_PORT`                      | Optional (default: `3306`)           | Database port number.                                                      |
| `DB_HOST`                      | Required                             | Database host.                                                             |
| `DB_NAME`                      | Optional (if provided via `-d` flag) | Database name.                                                             |
| `DB_USERNAME`                  | Required                             | Database username.                                                         |
| `DB_PASSWORD`                  | Required                             | Database password.                                                         |
| `DB_SSL_CA`                    | Optional                             | Database client CA certificate file                                        |
| `DB_SSL_MODE`                  | Optional(`0 or 1`) default: `0`      | Database client Enable CA validation                                       |
| `AWS_ACCESS_KEY`               | Required for S3 storage              | AWS S3 Access Key.                                                         |
| `AWS_SECRET_KEY`               | Required for S3 storage              | AWS S3 Secret Key.                                                         |
| `AWS_BUCKET_NAME`              | Required for S3 storage              | AWS S3 Bucket Name.                                                        |
| `AWS_REGION`                   | Required for S3 storage              | AWS Region.                                                                |
| `AWS_DISABLE_SSL`              | Optional                             | Disable SSL for S3 storage.                                                |
| `AWS_FORCE_PATH_STYLE`         | Optional                             | Force path-style access for S3 storage.                                    |
| `FILE_NAME`                    | Optional (if provided via `--file`)  | File name for restoration (e.g., `.sql`, `.sql.gz`).                       |
| `GPG_PASSPHRASE`               | Optional                             | GPG passphrase for encrypting/decrypting backups.                          |
| `GPG_PUBLIC_KEY`               | Optional                             | GPG public key for encrypting backups (e.g., `/config/public_key.asc`).    |
| `BACKUP_CRON_EXPRESSION`       | Optional (flag `-e`)                 | Cron expression for scheduled backups.                                     |
| `BACKUP_RETENTION_DAYS`        | Optional                             | Delete backups older than the specified number of days.                    |
| `BACKUP_CONFIG_FILE`           | Optional  (flag `-c`)                | Configuration file for multi database backup. (e.g: `/backup/config.yaml`) |
| `SSH_HOST`                     | Required for SSH storage             | SSH remote hostname or IP.                                                 |
| `SSH_USER`                     | Required for SSH storage             | SSH remote username.                                                       |
| `SSH_PASSWORD`                 | Optional                             | SSH remote user's password.                                                |
| `SSH_IDENTIFY_FILE`            | Optional                             | SSH remote user's private key.                                             |
| `SSH_PORT`                     | Optional (default: `22`)             | SSH remote server port.                                                    |
| `REMOTE_PATH`                  | Required for SSH/FTP storage         | Remote path (e.g., `/home/toto/backup`).                                   |
| `FTP_HOST`                     | Required for FTP storage             | FTP hostname.                                                              |
| `FTP_PORT`                     | Optional (default: `21`)             | FTP server port.                                                           |
| `FTP_USER`                     | Required for FTP storage             | FTP username.                                                              |
| `FTP_PASSWORD`                 | Required for FTP storage             | FTP user password.                                                         |
| `TARGET_DB_HOST`               | Required for migration               | Target database host.                                                      |
| `TARGET_DB_PORT`               | Optional (default: `5432`)           | Target database port.                                                      |
| `TARGET_DB_NAME`               | Required for migration               | Target database name.                                                      |
| `TARGET_DB_USERNAME`           | Required for migration               | Target database username.                                                  |
| `TARGET_DB_PASSWORD`           | Required for migration               | Target database password.                                                  |
| `TARGET_DB_URL`                | Optional                             | Target database URL in JDBC URI format.                                    |
| `TG_TOKEN`                     | Required for Telegram notifications  | Telegram token (`BOT-ID:BOT-TOKEN`).                                       |
| `TG_CHAT_ID`                   | Required for Telegram notifications  | Telegram Chat ID.                                                          |
| `TZ`                           | Optional                             | Time zone for scheduling.                                                  |
| `AZURE_STORAGE_CONTAINER_NAME` | Required for Azure Blob Storage      | Azure storage container name.                                              |
| `AZURE_STORAGE_ACCOUNT_NAME`   | Required for Azure Blob Storage      | Azure storage account name.                                                |
| `AZURE_STORAGE_ACCOUNT_KEY`    | Required for Azure Blob Storage      | Azure storage account key.                                                 |

---

## Scheduled Backups

### Running in Scheduled Mode

- **Docker**: Use the `--cron-expression` flag or the `BACKUP_CRON_EXPRESSION` environment variable to schedule backups.
- **Kubernetes**: Use a `CronJob` resource for scheduled backups.

### Cron Syntax

The cron syntax consists of five fields:

```conf
* * * * * command
```

| Field         | Description                  | Values         |
|---------------|------------------------------|----------------|
| Minute        | Minute of the hour           | `0-59`         |
| Hour          | Hour of the day              | `0-23`         |
| Day of Month  | Day of the month             | `1-31`         |
| Month         | Month of the year            | `1-12`         |
| Day of Week   | Day of the week (0 = Sunday) | `0-7`          |

#### Examples

- **Every 30 minutes**: `*/30 * * * *`
- **Every hour at minute 0**: `0 * * * *`
- **Every day at 1:00 AM**: `0 1 * * *`

### Predefined Schedules

| Entry                      | Description                                | Equivalent To |
|----------------------------|--------------------------------------------|---------------|
| `@yearly` (or `@annually`) | Run once a year, midnight, Jan. 1st        | `0 0 1 1 *`   |
| `@monthly`                 | Run once a month, midnight, first of month | `0 0 1 * *`   |
| `@weekly`                  | Run once a week, midnight between Sat/Sun  | `0 0 * * 0`   |
| `@daily` (or `@midnight`)  | Run once a day, midnight                   | `0 0 * * *`   |
| `@hourly`                  | Run once an hour, beginning of hour        | `0 * * * *`   |

### Intervals

You can also schedule backups at fixed intervals using the format:

```conf
@every <duration>
```

- Example: `@every 1h30m10s` runs the backup every 1 hour, 30 minutes, and 10 seconds.
