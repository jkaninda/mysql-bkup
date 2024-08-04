---
title: Restore database from SSH
layout: default
parent: How Tos
nav_order: 6
---
# Restore database from SSH remote server

To restore the database from your remote server, you need to add `restore` subcommand to `mysql-bkup` or `bkup` and specify the file to restore by adding `--file store_20231219_022941.sql.gz`.

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
    command:
      - /bin/sh
      - -c
      - mysql-bkup restore --storage ssh -d my-database -f store_20231219_022941.sql.gz --path /home/jkaninda/backups
    volumes:
      - ./backup:/backup
    environment:
      - DB_PORT=3306
      - DB_HOST=postgres
      - DB_NAME=database
      - DB_USERNAME=username
      - DB_PASSWORD=password
      ## SSH config
      - SSH_HOST_NAME="hostname"
      - SSH_PORT=22
      - SSH_USER=user
      - SSH_REMOTE_PATH=/home/jkaninda/backups
      - SSH_IDENTIFY_FILE=/tmp/id_ed25519
      ## We advise you to use a private jey instead of password
      #- SSH_PASSWORD=password
    # mysql-bkup container must be connected to the same network with your database
    networks:
      - web
networks:
  web:
```
## Restore on Kubernetes

Simple Kubernetes restore Job:

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: restore-db
spec:
  template:
    spec:
      containers:
        - name: mysql-bkup
          image: jkaninda/mysql-bkup
          command:
            - /bin/sh
            - -c
            - bkup restore -s ssh -f store_20231219_022941.sql.gz
          env:
            - name: DB_PORT
              value: "3306"
            - name: DB_HOST
              value: ""
            - name: DB_NAME
              value: ""
            - name: DB_USERNAME
              value: ""
            # Please use secret!
            - name: DB_PASSWORD
              value: ""
            - name: SSH_HOST_NAME
              value: ""
            - name: SSH_PORT
              value: "22"
            - name: SSH_USER
              value: "xxx"
            - name: SSH_REMOTE_PATH
              value: "/home/jkaninda/backups"
            - name: AWS_ACCESS_KEY
              value: "xxxx"
            - name: SSH_IDENTIFY_FILE
              value: "/tmp/id_ed25519"
      restartPolicy: Never
  backoffLimit: 4
```