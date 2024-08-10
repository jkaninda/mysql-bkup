---
title: Restore database
layout: default
parent: How Tos
nav_order: 4
---

# Restore database

To restore the database, you need to add `restore` command and specify the file to restore by adding `--file store_20231219_022941.sql.gz`.

{: .note }
It supports __.sql__ and __.sql.gz__ compressed file.

### Restore

```yml
services:
  mysql-bkup:
    # In production, it is advised to lock your image tag to a proper
    # release version instead of using `latest`.
    # Check https://github.com/jkaninda/mysql-bkup/releases
    # for a list of available releases.
    image: jkaninda/mysql-bkup
    container_name: mysql-bkup
    command: restore -d database -f store_20231219_022941.sql.gz
    volumes:
      - ./backup:/backup
    environment:
      - DB_PORT=3306
      - DB_HOST=mysql
      - DB_NAME=database
      - DB_USERNAME=username
      - DB_PASSWORD=password
    # mysql-bkup container must be connected to the same network with your database
    networks:
      - web
networks:
  web:
```