---
title: Backup to SSH
layout: default
parent: How Tos
nav_order: 3
---
# Backup to SSH remote server


As described for s3 backup section, to change the storage of your backup and use SSH Remote server as storage. You need to add `--storage ssh` or `--storage remote`.
You need to add the full remote path by adding `--path /home/jkaninda/backups` flag or using `SSH_REMOTE_PATH` environment variable.

{: .note }
These environment variables are required for SSH backup `SSH_HOST_NAME`, `SSH_USER`, `SSH_REMOTE_PATH`, `SSH_IDENTIFY_FILE`, `SSH_PORT` or `SSH_PASSWORD` if you dont use a private key to access to your server.
Accessing the remote server using password is not recommended, use private key instead.

```yml
services:
  mysql-bkup:
    # In production, it is advised to lock your image tag to a proper
    # release version instead of using `latest`.
    # Check https://github.com/jkaninda/mysql-bkup/releases
    # for a list of available releases.
    image: jkaninda/mysql-bkup
    container_name: mysql-bkup
    command: backup --storage remote -d database
    volumes:
      - ./id_ed25519:/tmp/id_ed25519"
    environment:
      - DB_PORT=3306
      - DB_HOST=mysql
      #- DB_NAME=database
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


### Recurring backups to SSH remote server

As explained above, you need just to add required environment variables and specify the storage type `--storage ssh`.
You can use `--mode scheduled` and specify the periodical backup time by adding `--period "0 1 * * *"` flag as described below.

```yml
services:
  mysql-bkup:
    # In production, it is advised to lock your image tag to a proper
    # release version instead of using `latest`.
    # Check https://github.com/jkaninda/mysql-bkup/releases
    # for a list of available releases.
    image: jkaninda/mysql-bkup
    container_name: mysql-bkup
    command: backup -d database --storage ssh --mode scheduled --period "0 1 * * *"
    volumes:
      - ./id_ed25519:/tmp/id_ed25519"
    environment:
      - DB_PORT=3306
      - DB_HOST=mysql
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

## Deploy on Kubernetes

For Kubernetes, you don't need to run it in scheduled mode.
You can deploy it as CronJob.

Simple Kubernetes CronJob usage:

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: bkup-job
spec:
  schedule: "0 1 * * *"
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: mysql-bkup
            image: jkaninda/mysql-bkup
            command:
            - /bin/sh
            - -c
            - mysql-bkup backup -s ssh
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
```