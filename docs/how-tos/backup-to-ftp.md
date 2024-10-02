---
title: Backup to FTP remote server
layout: default
parent: How Tos
nav_order: 4
---
# Backup to FTP remote server


As described for SSH backup section, to change the storage of your backup and use FTP Remote server as storage. You need to add `--storage ftp`.
You need to add the full remote path by adding `--path /home/jkaninda/backups` flag or using `REMOTE_PATH` environment variable.

{: .note }
These environment variables are required for SSH backup `FTP_HOST`, `FTP_USER`, `REMOTE_PATH`, `FTP_PORT` or `FTP_PASSWORD`.

```yml
services:
  mysql-bkup:
    # In production, it is advised to lock your image tag to a proper
    # release version instead of using `latest`.
    # Check https://github.com/jkaninda/mysql-bkup/releases
    # for a list of available releases.
    image: jkaninda/mysql-bkup
    container_name: mysql-bkup
    command: backup --storage ftp -d database
    environment:
      - DB_PORT=3306
      - DB_HOST=postgres
      - DB_NAME=database
      - DB_USERNAME=username
      - DB_PASSWORD=password
      ## FTP config
      - FTP_HOST="hostname"
      - FTP_PORT=21
      - FTP_USER=user
      - FTP_PASSWORD=password
      - REMOTE_PATH=/home/jkaninda/backups

    # pg-bkup container must be connected to the same network with your database
    networks:
      - web
networks:
  web:
```