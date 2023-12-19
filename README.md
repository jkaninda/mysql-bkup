# MySQL Backup
MySQL Backup tool, backup database to S3 or Object Storage

- Docker
- Kubernetes

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
| mysql_bkup    | bkup   | CLI utility                    |
| --operation   | -o     | Set operation. backup or restore (default: backup)    |
| --destination | -d     | Set destination. local or s3 (default: local)   |
| --source      | -s     | Set source. local or s3 (default: local)        |
| --file        | -f     | Set file name for restoration      |
| --database        | -db     | Set database name      |
| --port        | -p     | Set database port (default: 3306)      |
| --timeout     | -t     | Set timeout (default: 60s)        |
| --help        | -h     | Print this help message and exit   |
| --version     | -V     | Print version information and exit |

## Backup database :

Simple backup usage

```sh
bkup --operation backup
```
```sh
bkup -o backup
```
### S3

```sh
bkup --operation backup --destination s3
```
## Docker run:

```sh
docker run --rm --network your_network_name --name mysql-bkup -v $PWD/backup:/backup/ -e "DB_HOST=database_host_name" -e "DB_USERNAME=username" -e "DB_PASSWORD=password" jkaninda/mysql-bkup:latest  bkup -o backup -db database_name
```

## Docker compose file:
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

Simple database restore operation usage

```sh
bkup --operation restore --file database_20231217_115621.sql 
```

```sh
bkup -o restore -f database_20231217_115621.sql 
```
### S3

```sh
bkup --operation restore --source s3 --file database_20231217_115621.sql 
```

## Docker run:

```sh
docker run --rm --network your_network_name --name mysql-bkup -v $PWD/backup:/backup/ -e "DB_HOST=database_host_name" -e "DB_USERNAME=username" -e "DB_PASSWORD=password" jkaninda/mysql-bkup:latest  bkup -o backup -db database_name -f napata_20231219_022941.sql.gz
```

## Docker compose file:

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

```sh
bkup --operation backup --destination s3 -database mydatabase 
```
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
## Run "docker run" from crontab

Make an automated backup (every night at 1).

> backup_script.sh

```sh
#!/bin/sh
DB_USERNAME='db_username'
DB_PASSWORD='password'
DB_HOST='db_hostname'
DB_NAME='db_name'
BACKUP_DIR='/some/path/backup/'

docker run --rm --name mysql-bkup -v $BACKUP_DIR:/backup/ -e "DB_HOST=$DB_HOST" -e "DB_USERNAME=$DB_USERNAME" -e "DB_PASSWORD=$DB_PASSWORD" jkaninda/mysql-bkup:latest  bkup -o backup -db $DB_NAME
```

```sh
chmod +x backup_script.sh
```

Your crontab looks like this:

```conf
0 1 * * * /path/to/backup_script.sh
```

## Kubernetes CronJob

Simple Kubernetes CronJob usage:

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