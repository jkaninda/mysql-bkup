---
title: Encrypt backups using GPG
layout: default
parent: How Tos
nav_order: 7
---
# Encrypt backup

The image supports encrypting backups using GPG out of the box. In case a `GPG_PASSPHRASE` environment variable is set, the backup archive will be encrypted using the given key and saved as a sql.gpg file instead or sql.gz.gpg.

{: .warning }
To restore an encrypted backup, you need to provide the same GPG passphrase or key used during backup process.

- GPG home directory `/config/gnupg`
- Cipher algorithm `aes256`
- 
To decrypt manually, you need to install `gnupg`

### Decrypt backup

```shell
gpg --batch --passphrase "my-passphrase" \
--output database_20240730_044201.sql.gz \
--decrypt database_20240730_044201.sql.gz.gpg
```

### Backup

```yml
services:
  mysql-bkup:
    # In production, it is advised to lock your image tag to a proper
    # release version instead of using `latest`.
    # Check https://github.com/jkaninda/mysql-bkup/releases
    # for a list of available releases.
    image: jkaninda/mysql-bkup
    container_name: mysql-bkup
    command: backup -d database
    volumes:
      - ./backup:/backup
    environment:
      - DB_PORT=3306
      - DB_HOST=mysql
      - DB_NAME=database
      - DB_USERNAME=username
      - DB_PASSWORD=password
      ## Required to encrypt backup
      - GPG_PASSPHRASE=my-secure-passphrase
    # mysql-bkup container must be connected to the same network with your database
    networks:
      - web
networks:
  web:
```