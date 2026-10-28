---
title: Encrypt backups using GPG
layout: default
parent: How Tos
nav_order: 8
---
# Encrypt Backup

The image supports encrypting backups using one of two methods: **GPG with a passphrase** or **GPG with a public key**. When a `GPG_PASSPHRASE` or `GPG_PUBLIC_KEY` environment variable is set, the backup archive will be encrypted and saved as a `.sql.gpg` or `.sql.gz.gpg` file.

{: .warning }
To restore an encrypted backup, you must provide the same GPG passphrase or private key used during the backup process.

---

## Key Features

- **Cipher Algorithm**: `aes256`
- **Automatic Restoration**: Backups encrypted with a GPG passphrase can be restored automatically without manual decryption.
- **Manual Decryption**: Backups encrypted with a GPG public key require manual decryption before restoration.

---

## Using GPG Passphrase

To encrypt backups using a GPG passphrase, set the `GPG_PASSPHRASE` environment variable. The backup will be encrypted and can be restored automatically.

### Example Configuration

```yaml
services:
  mysql-bkup:
    # In production, lock your image tag to a specific release version
    # instead of using `latest`. Check https://github.com/jkaninda/mysql-bkup/releases
    # for available releases.
    image: jkaninda/mysql-bkup
    container_name: mysql-bkup
    command: backup -d database
    volumes:
      - ./backup:/backup
    environment:
      - DB_PORT=3306
      - DB_HOST=mysql
      - DB_NAME=database
      - DB_USER=username
      - DB_PASSWORD=password
      ## Required to encrypt backup
      - GPG_PASSPHRASE=my-secure-passphrase
    # Ensure the pg-bkup container is connected to the same network as your database
    networks:
      - web

networks:
  web:
```

---

## Using GPG Public Key

To encrypt backups using a GPG public key, set the `GPG_PUBLIC_KEY` environment variable to the path of your public key file. To restore such backups, set `GPG_PRIVATE_KEY` (a file path or the inline armored key; `/config/gnupg/private_key.asc` is picked up automatically) and, if the key is protected, `GPG_PRIVATE_KEY_PASSPHRASE` (defaults to `GPG_PASSPHRASE`).

### Example Configuration

```yaml
services:
  mysql-bkup:
    # In production, lock your image tag to a specific release version
    # instead of using `latest`. Check https://github.com/jkaninda/mysql-bkup/releases
    # for available releases.
    image: jkaninda/mysql-bkup
    container_name: mysql-bkup
    command: backup -d database
    volumes:
      - ./backup:/backup
      - ./public_key.asc:/config/public_key.asc
    environment:
      - DB_PORT=3306
      - DB_HOST=mysql
      - DB_NAME=database
      - DB_USER=username
      - DB_PASSWORD=password
      ## Required to encrypt backup
      - GPG_PUBLIC_KEY=/config/public_key.asc
    # Ensure the pg-bkup container is connected to the same network as your database
    networks:
      - web

networks:
  web:
```

---

## Manual Decryption

If you encrypted your backup using a GPG public key, you must manually decrypt it before restoration. Use the `gnupg` tool for decryption.

### Decrypt Using a Passphrase

```bash
gpg --batch --passphrase "my-passphrase" \
  --output database_20240730_044201.sql.gz \
  --decrypt database_20240730_044201.sql.gz.gpg
```

### Decrypt Using a Private Key

```bash
gpg --output database_20240730_044201.sql.gz \
  --decrypt database_20240730_044201.sql.gz.gpg
```

---

## Key Notes

- **Automatic Restoration**: Backups encrypted with a GPG passphrase can be restored directly without manual decryption.
- **Private key restore**: Backups encrypted with a GPG public key restore directly when `GPG_PRIVATE_KEY` is set; manual decryption also works.
- **Fail closed**: if `GPG_PUBLIC_KEY` (or any configured key) cannot be read, the backup fails instead of being stored unencrypted.
- **Security**: Always keep your GPG passphrase and private key secure. Use Kubernetes Secrets or other secure methods to manage sensitive data.

---

## Using age

[age](https://age-encryption.org) is a modern alternative to GPG. Set `AGE_RECIPIENTS` (one or more `age1...` or `ssh-ed25519 ...` public keys, comma or newline separated, or `AGE_RECIPIENTS_FILE`) and backups are saved as `.sql.gz.age`. Restore with the matching identity in `AGE_IDENTITY` (`AGE-SECRET-KEY-...`) or `AGE_IDENTITY_FILE` (default `/config/age/identity.txt`).

```yaml
    environment:
      - AGE_RECIPIENTS=age1ql3z7hjy54pw3hyww5ayyfg7zqgvc7w3j2elw8zmrj2kg5sfn9aqmcac8p
      # restore:
      # - AGE_IDENTITY_FILE=/run/secrets/age_identity
```

`AGE_PASSPHRASE` encrypts with a passphrase instead (it needs about 256 MiB of memory for the key derivation). `ENCRYPTION=auto|none|gpg|age` chooses explicitly; with `auto` (default) GPG wins when both GPG and age keys are set. Decrypt by hand with `age -d -i key.txt file.sql.gz.age | gunzip`.
