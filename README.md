# MySQL Backup
MySQL Backup tool, backup database to S3 or Object Storage

[![Build](https://github.com/jkaninda/mysql-bkup/actions/workflows/build.yml/badge.svg)](https://github.com/jkaninda/mysql-bkup/actions/workflows/build.yml)
![Docker Image Size (latest by date)](https://img.shields.io/docker/image-size/jkaninda/mysql-bkup?style=flat-square)
![Docker Pulls](https://img.shields.io/docker/pulls/jkaninda/mysql-bkup?style=flat-square)

> Runs on:
- Docker
- Kubernetes

> Links:
- [Docker Hub](https://hub.docker.com/r/jkaninda/mysql-bkup)
- [Github](https://github.com/jkaninda/mysql-bkup)

> Postgres solution :
- [Postgress](https://github.com/jkaninda/pg-bkup)



## Storage:
- local
- s3
- Object storage
## Usage

| Options       | Shorts | Usage                              |
|---------------|--------|------------------------------------|
| mysql_bkup    | bkup   | CLI utility                    |
| --operation   | -o     | Set operation. backup or restore (default: backup)    |
| --storage      | -s     | Set storage. local or s3 (default: local)        |
| --file        | -f     | Set file name for restoration      |
| --path        |      | Set s3 path without file name. eg: /custom_path      |
| --dbname        | -d     | Set database name      |
| --port        | -p     | Set database port (default: 3306)      |
| --mode     | -m     | Set execution mode. default or scheduled (default: default)        |
| --period     |      | Set crontab period for scheduled mode only. (default: "*/30 * * * *")        |
| --timeout     | -t     | Set timeout (default: 60s)        |
| --help        | -h     | Print this help message and exit   |
| --version     | -V     | Print version information and exit |

## Backup database :

Simple backup usage

```sh
bkup --operation backup --dbname database_name
```
```sh
bkup -o backup -d database_name
```
### S3

```sh
bkup --operation backup --storage s3 --dbname database_name
```
## Docker run:

```sh
docker run --rm --network your_network_name --name mysql-bkup -v $PWD/backup:/backup/ -e "DB_HOST=database_host_name" -e "DB_USERNAME=username" -e "DB_PASSWORD=password" jkaninda/mysql-bkup:latest  bkup -o backup -d database_name
```

## Docker compose file:
```yaml
version: '3'
services:
  mariadb:
    container_name: mariadb
    image: mariadb
    environment:
      MYSQL_DATABASE: mariadb
      MYSQL_USER: mariadb
      MYSQL_PASSWORD: password
      MYSQL_ROOT_PASSWORD: password
  mysql-bkup:
    image: jkaninda/mysql-bkup
    container_name: mysql-bkup
    command:
      - /bin/sh
      - -c
      - bkup --operation backup -d database_name
    volumes:
      - ./backup:/backup
    environment:
      - DB_PORT=3306
      - DB_HOST=mariadb
      - DB_USERNAME=mariadb
      - DB_PASSWORD=password
```
## Restore database :

Simple database restore operation usage

```sh
bkup --operation restore --dbname database_name --file database_20231217_115621.sql
```

```sh
bkup -o restore -f database_20231217_115621.sql 
```
### S3

```sh
bkup --operation restore --storage s3 --file database_20231217_115621.sql 
```

## Docker run:

```sh
docker run --rm --network your_network_name --name mysql-bkup -v $PWD/backup:/backup/ -e "DB_HOST=database_host_name" -e "DB_USERNAME=username" -e "DB_PASSWORD=password" jkaninda/mysql-bkup  bkup -o backup -d database_name -f mydb_20231219_022941.sql.gz
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
    image: jkaninda/mysql-bkup
    container_name: mysql-bkup
    command:
      - /bin/sh
      - -c
      - bkup --operation restore --file database_20231217_115621.sql --dbname database_name
    volumes:
      - ./backup:/backup
    environment:
      #- FILE_NAME=mariadb_20231217_040238.sql # Optional if file name is set from command
      - DB_PORT=3306
      - DB_HOST=mariadb
      - DB_NAME=mariadb
      - DB_USERNAME=mariadb
      - DB_PASSWORD=password
```
## Run 

```sh
docker-compose up -d
```
## Backup to S3

```sh
docker run --rm --privileged --device /dev/fuse --name mysql-bkup -e "DB_HOST=db_hostname" -e "DB_USERNAME=username" -e "DB_PASSWORD=password" -e "ACCESS_KEY=your_access_key" -e "SECRET_KEY=your_secret_key" -e "BUCKETNAME=your_bucket_name" -e "S3_ENDPOINT=https://eu2.contabostorage.com" jkaninda/mysql-bkup  bkup -o backup -s s3 -d database_name
```
> To change s3 backup path add this flag : --path myPath . default path is /mysql_bkup

Simple S3 backup usage

```sh
bkup --operation backup --storage s3 --dbname mydatabase 
```
```yaml
version: '3'
services:
  mysql-bkup:
    image: jkaninda/mysql-bkup
    container_name: mysql-bkup
    tty: true
    privileged: true
    devices:
    - "/dev/fuse"
    command:
      - /bin/sh
      - -c
      - mysql_bkup --operation restore --storage s3 -f database_20231217_115621.sql.gz
    environment:
      - DB_PORT=3306
      - DB_HOST=mysql
      - DB_NAME=mariadb
      - DB_USERNAME=mariadb
      - DB_PASSWORD=password
      - ACCESS_KEY=${ACCESS_KEY}
      - SECRET_KEY=${SECRET_KEY}
      - BUCKETNAME=${BUCKETNAME}
      - S3_ENDPOINT=${S3_ENDPOINT}

```
## Run in Scheduled mode

This tool can be run as CronJob in Kubernetes for a regular backup which makes deployment on Kubernetes easy as Kubernetes has CronJob resources.
For Docker, you need to run it in scheduled mode by adding `--mode scheduled` flag and specify the periodical backup time by adding `--period "*/30 * * * *"` flag.

Make an automated backup on Docker

## Syntax of crontab (field description)

The syntax is:

- 1: Minute (0-59)
- 2: Hours (0-23)
- 3: Day (0-31)
- 4: Month (0-12 [12 == December])
- 5: Day of the week(0-7 [7 or 0 == sunday])

Easy to remember format:

```conf
* * * * * command to be executed
```

```conf
- - - - -
| | | | |
| | | | ----- Day of week (0 - 7) (Sunday=0 or 7)
| | | ------- Month (1 - 12)
| | --------- Day of month (1 - 31)
| ----------- Hour (0 - 23)
------------- Minute (0 - 59)
```

## Example of scheduled mode

> Docker run :

```sh
docker run --rm --name mysql-bkup -v $BACKUP_DIR:/backup/ -e "DB_HOST=$DB_HOST" -e "DB_USERNAME=$DB_USERNAME" -e "DB_PASSWORD=$DB_PASSWORD" jkaninda/mysql-bkup:latest  bkup --operation backup --dbname $DB_NAME --mode scheduled --period "*/30 * * * *"
```

> With Docker compose

```yaml
version: "3"
services:
  mysql-bkup:
    image: jkaninda/mysql-bkup:v0.3
    container_name: mysql-bkup
    command:
      - /bin/sh
      - -c
      - bkup --operation backup --storage s3 --path /mys3_custome_path --dbname database_name --mode scheduled --period "*/30 * * * *"
    environment:
      - DB_PORT=3306
      - DB_HOST=mysqlhost
      - DB_USERNAME=userName
      - DB_PASSWORD=${DB_PASSWORD}
      - ACCESS_KEY=${ACCESS_KEY}
      - SECRET_KEY=${SECRET_KEY}
      - BUCKETNAME=${BUCKETNAME}
      - S3_ENDPOINT=${S3_ENDPOINT}
```


## Kubernetes CronJob
For Kubernetes you don't need to run it in scheduled mode.

Simple Kubernetes CronJob usage:

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: mysql-bkup-job
spec:
  schedule: "0 1 * * *"
  jobTemplate:
    spec:
      template:
        spec:
          backoffLimit: 2
          containers:
          - name: mysql-bkup
            image: jkaninda/mysql-bkup
            securityContext:
              privileged: true
            command:
            - /bin/sh
            - -c
            - bkup -o backup -s s3 --path /custom_path
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
                value: "password"
              - name: ACCESS_KEY
                value: ""
              - name: SECRET_KEY
                value: ""
              - name: BUCKETNAME
                value: ""
              - name: S3_ENDPOINT
                value: "https://s3.amazonaws.com"
          restartPolicy: Never
```