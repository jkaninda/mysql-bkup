---
title: Overview
layout: home
nav_order: 1
---

# About mysql-bkup
{:.no_toc}
MySQL Backup is a Docker container image that can be used to backup, restore and migrate MySQL database. It supports local storage, AWS S3 or any S3 Alternatives for Object Storage, and SSH remote storage.
It also supports __encrypting__ your backups using GPG.

We are open to receiving stars, PRs, and issues!


{: .fs-6 .fw-300 }

---

The [jkaninda/mysql-bkup](https://hub.docker.com/r/jkaninda/mysql-bkup) Docker image can be deployed on Docker, Docker Swarm and Kubernetes. 
It handles __recurring__ backups of postgres database on Docker and can be deployed as __CronJob on Kubernetes__ using local, AWS S3 or SSH compatible storage.

It also supports database __encryption__ using GPG.


{: .note }
Code and documentation for `v1` version on [this branch][v1-branch].

[v1-branch]: https://github.com/jkaninda/mysql-bkup

---

## Quickstart

### Simple backup using Docker CLI

To run a one time backup, bind your local volume to `/backup` in the container and run the `backup` command:

```shell
 docker run --rm --network your_network_name \
 -v $PWD/backup:/backup/ \
 -e "DB_HOST=dbhost" \
 -e "DB_USERNAME=username" \
 -e "DB_PASSWORD=password" \
 jkaninda/mysql-bkup backup -d database_name
```

Alternatively, pass a `--env-file` in order to use a full config as described below.

```yaml
 docker run --rm --network your_network_name \
 --env-file your-env-file \
 -v $PWD/backup:/backup/ \
 jkaninda/mysql-bkup backup -d database_name
```

### Simple backup in docker compose file

```yaml
services:
  mysql-bkup:
    # In production, it is advised to lock your image tag to a proper
    # release version instead of using `latest`.
    # Check https://github.com/jkaninda/mysql-bkup/releases
    # for a list of available releases.
    image: jkaninda/mysql-bkup
    container_name: mysql-bkup
    command: backup
    volumes:
      - ./backup:/backup
    environment:
      - DB_PORT=3306
      - DB_HOST=mysql
      - DB_NAME=foo
      - DB_USERNAME=bar
      - DB_PASSWORD=password
    # mysql-bkup container must be connected to the same network with your database
    networks:
       - web
networks:
  web:
```
## Kubernetes

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: backup-job
spec:
  ttlSecondsAfterFinished: 100
  template:
    spec:
      containers:
        - name: mysql-bkup
          # In production, it is advised to lock your image tag to a proper
          # release version instead of using `latest`.
          # Check https://github.com/jkaninda/mysql-bkup/releases
          # for a list of available releases.
          image: jkaninda/mysql-bkup
          command:
            - /bin/sh
            - -c
            - backup -d dbname
          resources:
            limits:
              memory: "128Mi"
              cpu: "500m"
          env:
            - name: DB_HOST
              value: "mysql"
            - name: DB_USERNAME
              value: "user"
            - name: DB_PASSWORD
              value: "password"
          volumeMounts:
            - mountPath: /backup
              name: backup
      volumes:
        - name: backup
          hostPath:
            path: /home/toto/backup # directory location on host
            type: Directory # this field is optional
      restartPolicy: Never
```

## Available image registries

This Docker image is published to both Docker Hub and the GitHub container registry.
Depending on your preferences and needs, you can reference both `jkaninda/mysql-bkup` as well as `ghcr.io/jkaninda/mysql-bkup`:

```
docker pull jkaninda/mysql-bkup
docker pull ghcr.io/jkaninda/mysql-bkup
```

Documentation references Docker Hub, but all examples will work using ghcr.io just as well.

## Supported Engines

This image is developed and tested against the Docker CE engine and Kubernetes exclusively.
While it may work against different implementations, there are no guarantees about support for non-Docker engines.

## References

We decided to publish this image as a simpler and more lightweight alternative because of the following requirements:

- The original image is based on `ubuntu` and requires additional tools, making it heavy.
- This image is written in Go.
- `arm64` and `arm/v7` architectures are supported.
- Docker in Swarm mode is supported.
- Kubernetes is supported.
