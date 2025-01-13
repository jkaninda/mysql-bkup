---
title: Quickstart
layout: home
nav_order: 2
---

# Quickstart

This guide provides quick examples for running backups using Docker CLI, Docker Compose, and Kubernetes.

---

## Simple Backup Using Docker CLI

To run a one-time backup, bind your local volume to `/backup` in the container and execute the `backup` command:

```bash
docker run --rm --network your_network_name \
  -v $PWD/backup:/backup/ \
  -e "DB_HOST=dbhost" \
  -e "DB_USERNAME=username" \
  -e "DB_PASSWORD=password" \
  jkaninda/mysql-bkup backup -d database_name
```

### Using an Environment File

Alternatively, you can use an `--env-file` to pass a full configuration:

```bash
docker run --rm --network your_network_name \
  --env-file your-env-file \
  -v $PWD/backup:/backup/ \
  jkaninda/mysql-bkup backup -d database_name
```

---

## Simple Backup Using Docker Compose

Below is an example `docker-compose.yml` configuration for running a backup:

```yaml
services:
  mysql-bkup:
    # In production, lock the image tag to a specific release version.
    # Check https://github.com/jkaninda/mysql-bkup/releases for available releases.
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
      - TZ=Europe/Paris
    # Ensure the mysql-bkup container is connected to the same network as your database.
    networks:
      - web

networks:
  web:
```

---

## Recurring Backup with Docker

To schedule recurring backups, use the `--cron-expression` flag:

```bash
docker run --rm --network network_name \
  -v $PWD/backup:/backup/ \
  -e "DB_HOST=hostname" \
  -e "DB_USERNAME=user" \
  -e "DB_PASSWORD=password" \
  jkaninda/mysql-bkup backup -d dbName --cron-expression "@every 15m"
```

For predefined schedules, refer to the [documentation](https://jkaninda.github.io/mysql-bkup/reference/#predefined-schedules).

---

## Backup Using Kubernetes

Below is an example Kubernetes `Job` configuration for running a backup:

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
          # In production, lock the image tag to a specific release version.
          # Check https://github.com/jkaninda/mysql-bkup/releases for available releases.
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
              value: "postgres"
            - name: DB_PASSWORD
              value: "password"
          volumeMounts:
            - mountPath: /backup
              name: backup
      volumes:
        - name: backup
          hostPath:
            path: /home/toto/backup  # Directory location on the host
            type: Directory  # Optional field
      restartPolicy: Never
```

---

## Key Notes

- **Volume Binding**: Ensure the `/backup` directory is mounted to persist backup files.
- **Environment Variables**: Use environment variables or an `--env-file` to pass database credentials and other configurations.
- **Cron Expressions**: Use standard cron expressions or predefined schedules for recurring backups.
- **Kubernetes Jobs**: Use Kubernetes `Job` or `CronJob` for running backups in a Kubernetes cluster.