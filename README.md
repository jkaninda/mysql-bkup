# MySQL Backup
MySQL Backup docker container image

[![Build](https://github.com/jkaninda/mysql-bkup/actions/workflows/build.yml/badge.svg)](https://github.com/jkaninda/mysql-bkup/actions/workflows/build.yml)
![Docker Image Size (latest by date)](https://img.shields.io/docker/image-size/jkaninda/mysql-bkup?style=flat-square)
![Docker Pulls](https://img.shields.io/docker/pulls/jkaninda/mysql-bkup?style=flat-square)

- [Docker Hub](https://hub.docker.com/r/jkaninda/mysql-bkup)
- [Github](https://github.com/jkaninda/mysql-bkup)

## Storage:
- local
- s3

## Backup database :
```yaml
version: '3'
services:
  mariadb:
    container_name: mariadb
    image: mariadb:latest
    environment:
      MYSQL_DATABASE: mariadb
      MYSQL_USER: mariadb
      MYSQL_PASSWORD: password
      MYSQL_ROOT_PASSWORD: password
  mysql-bkup:
    image: jkaninda/mysql-bkup:latest
    container_name: mysql-bkup
    command:
     - /bin/sh
     - -c
     - backup
    volumes:
      - ./backup:/backup
    environment:
      - DB_PORT=3306
      - DB_HOST=mariadb
      - DB_DATABASE=mariadb
      - DB_USERNAME=mariadb
      - DB_PASSWORD=password
```
## Restore database :
```yaml
version: '3'
services:
  mariadb:
    container_name: mariadb
    image: mariadb:latest
    environment:
      MYSQL_DATABASE: mariadb
      MYSQL_USER: mariadb
      MYSQL_PASSWORD: password
      MYSQL_ROOT_PASSWORD: password
  mysql-bkup:
    image: jkaninda/mysql-bkup:latest
    container_name: mysql-bkup
    command: ["restore"]
    volumes:
      - ./backup:/backup
    environment:
      - FILE_NAME=mariadb_20231217_040238.sql
      - DB_PORT=3306
      - DB_HOST=mariadb
      - DB_DATABASE=mariadb
      - DB_USERNAME=mariadb
      - DB_PASSWORD=password
```
## Run 
```sh
docker-compose up -d
```
## Run on Kubernetes

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: mysql-bkup-job
spec:
  schedule: "0 0 * * *"
  jobTemplate:
    spec:
      template:
        spec:
          backoffLimit: 4
          containers:
          - name: mysql-bkup
            image: jkaninda/mysql-bkup:latest
            command:
            - /bin/sh
            - -c
            - backup; 
            env:
              - name: DB_PORT
                value: "3306"
              - name: DB_HOST
                value: "mysql-svc"
              - name: DB_DATABASE
                value: "mariadb"
              - name: DB_USERNAME
                value: "mariadb"
              # Please use secret!
              - name: DB_PASSWORD
                value: "password"
          restartPolicy: Never
```
