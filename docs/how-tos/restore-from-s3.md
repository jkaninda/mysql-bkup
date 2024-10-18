---
title: Restore database from AWS S3
layout: default
parent: How Tos
nav_order: 6
---

# Restore database from S3 storage

To restore the database, you need to add `restore` command and specify the file to restore by adding `--file store_20231219_022941.sql.gz`.

{: .note }
It supports __.sql__,__.sql.gpg__  and __.sql.gz__,__.sql.gz.gpg__ compressed file.

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
    command: restore --storage s3 -d my-database -f store_20231219_022941.sql.gz --path /my-custom-path
    volumes:
      - ./backup:/backup
    environment:
      - DB_PORT=3306
      - DB_HOST=mysql
      - DB_NAME=database
      - DB_USERNAME=username
      - DB_PASSWORD=password
      ## AWS configurations
      - AWS_S3_ENDPOINT=https://s3.amazonaws.com
      - AWS_S3_BUCKET_NAME=backup
      - AWS_REGION="us-west-2"
      - AWS_ACCESS_KEY=xxxx
      - AWS_SECRET_KEY=xxxxx
      ## In case you are using S3 alternative such as Minio and your Minio instance is not secured, you change it to true
      - AWS_DISABLE_SSL="false"
      - AWS_FORCE_PATH_STYLE="false"
    # mysql-bkup container must be connected to the same network with your database
    networks:
      - web
networks:
  web:
```
