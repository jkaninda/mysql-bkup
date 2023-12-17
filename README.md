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
- Object storage
## Usage

| Options       | Shorts | Usage                              |
|---------------|--------|------------------------------------|
| mysql_bkup    | bkup   | Command utility                    |
| --operation   | -o     | Set operation (default: backup)    |
| --destination | -d     | Set destination (default: local)   |
| --source      | -s     | Set source (default: local)        |
| --file        | -f     | Set file name for restoration      |
| --source      | -s     | Set source (default: local)        |
| --database        | -db     | Set database name      |
| --port        | -p     | Set database port (default: 3306)      |
| --timeout     | -t     | Set timeout (default: 120s)        |
| --help        | -h     | Print this help message and exit   |
| --version     | -V     | Print version information and exit |

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
      - bkup --operation backup -db mariadb
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
    command:
      - /bin/sh
      - -c
      - bkup --operation restore --file database_20231217_115621.sql
    volumes:
      - ./backup:/backup
    environment:
      #- FILE_NAME=mariadb_20231217_040238.sql # Optional if file name is set from command
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
## Backup to S3
Simple S3 backup usage
```yaml
  mysql-bkup:
    image: jkaninda/mysql-bkup:latest
    container_name: mysql-bkup
    tty: true
    privileged: true
    devices:
    - "/dev/fuse"
    command:
      - /bin/sh
      - -c
      - mysql_bkup --operation restore --source s3 -f database_20231217_115621.sql.gz
    volumes:
      - ./backup:/backup
    environment:
      - DB_PORT=3306
      - DB_HOST=mysql
      - DB_DATABASE=mariadb
      - DB_USERNAME=mariadb
      - DB_PASSWORD=password
      - ACCESS_KEY=${ACCESS_KEY}
      - SECRET_KEY=${SECRET_KEY}
      - BUCKETNAME=${BUCKETNAME}
      - S3_ENDPOINT=${S3_ENDPOINT}

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
            - bkup --operation backup 
            env:
              - name: DB_PORT
                value: "3306"
              - name: DB_HOST
                value: "mysql-svc"
              - name: DB_DATABASE
                value: "mariadb"
              - name: DB_USERNAME
                value: "mariadb"
              # Please use secret instead!
              - name: DB_PASSWORD
                value: "password"
          restartPolicy: Never
```