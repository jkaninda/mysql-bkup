---
layout: page
title: Old version
permalink: /old-version/
---

This is the documentation of mysql-backup for all old versions bellow `v1.0`.
In the old version, S3 storage was mounted using s3fs, so we decided to migrate to the official AWS SDK.

## Storage:
- local
- s3
- Object storage

## Volumes:

- /s3mnt => S3 mounting path
- /backup => local storage mounting path

### Usage

| Options               | Shorts | Usage                                                                  |
|-----------------------|--------|------------------------------------------------------------------------|
| mysql-bkup            | bkup   | CLI utility                                                            |
| backup                |        | Backup database operation                                              |
| restore               |        | Restore database operation                                             |
| history               |        | Show the history of backup                                             |
| --storage             | -s     | Storage. local or s3 (default: local)                                  |
| --file                | -f     | File name to restore                                                   |
| --path                |        | S3 path without file name. eg: /custom_path                            |
| --dbname              | -d     | Database name                                                          |
| --port                | -p     | Database port (default: 3306)                                          |
| --mode                | -m     | Execution mode. default or scheduled (default: default)                |
| --disable-compression |        | Disable database backup compression                                    |
| --prune               |        | Delete old backup, default disabled                                    |
| --keep-last           |        | Delete old backup created more than specified days ago, default 7 days |
| --period              |        | Crontab period for scheduled mode only. (default: "0 1 * * *")         |
| --help                | -h     | Print this help message and exit                                       |
| --version             | -V     | Print version information and exit                                     |


## Environment variables

| Name        | Requirement                                      | Description                                          |
|-------------|--------------------------------------------------|------------------------------------------------------|
| DB_PORT     | Optional, default 3306                           | Database port number                                 |
| DB_HOST     | Required                                         | Database host                                        |
| DB_NAME     | Optional if it was provided from the -d flag     | Database name                                        |
| DB_USERNAME | Required                                         | Database user name                                   |
| DB_PASSWORD | Required                                         | Database password                                    |
| ACCESS_KEY  | Optional, required for S3 storage                | AWS S3 Access Key                                    |
| SECRET_KEY  | Optional, required for S3 storage                | AWS S3 Secret Key                                    |
| BUCKET_NAME | Optional, required for S3 storage                | AWS S3 Bucket Name                                   |
| S3_ENDPOINT | Optional, required for S3 storage                | AWS S3 Endpoint                                      |
| FILE_NAME   | Optional if it was provided from the --file flag | Database file to restore (extensions: .sql, .sql.gz) |


## Note:

Creating a user for backup tasks who has read-only access is recommended!

> create read-only user


## Backup database :

Simple backup usage

```sh
bkup backup
```

### S3

```sh
mysql-bkup backup --storage s3
```
## Docker run:

```sh
docker run --rm --network your_network_name \
--name mysql-bkup -v $PWD/backup:/backup/ \
-e "DB_HOST=database_host_name" \
-e "DB_USERNAME=username" \
-e "DB_PASSWORD=password" jkaninda/mysql-bkup:v0.7  mysql-bkup backup -d database_name
```

## Docker compose file:
```yaml
version: '3'
services:
  postgres:
    image: postgres:14.5
    container_name: postgres
    restart: unless-stopped
    volumes:
      - ./postgres:/var/lib/postgresql/data
    environment:
      POSTGRES_DB: bkup
      POSTGRES_PASSWORD: password
      POSTGRES_USER: bkup
  mysql-bkup:
    image: jkaninda/mysql-bkup:v0.7
    container_name: mysql-bkup
    depends_on:
      - postgres
    command:
      - /bin/sh
      - -c
      - mysql-bkup backup -d bkup
    volumes:
      - ./backup:/backup
    environment:
      - DB_PORT=3306
      - DB_HOST=postgres
      - DB_NAME=bkup
      - DB_USERNAME=bkup
      - DB_PASSWORD=password
```
## Restore database :

Simple database restore operation usage

```sh
mysql-bkup restore --file database_20231217_115621.sql  --dbname database_name
```

```sh
mysql-bkup restore -f database_20231217_115621.sql -d database_name
```
### S3

```sh
mysql-bkup restore --storage s3 --file database_20231217_115621.sql --dbname database_name
```

## Docker run:

```sh
docker run --rm --network your_network_name \
--name mysql-bkup \
-v $PWD/backup:/backup/ \
-e "DB_HOST=database_host_name" \
-e "DB_USERNAME=username" \
-e "DB_PASSWORD=password" \
jkaninda/mysql-bkup:v0.7   mysql-bkup restore -d database_name -f store_20231219_022941.sql.gz
```

## Docker compose file:

```yaml
version: '3'
services:
  mysql-bkup:
    image: jkaninda/mysql-bkup:v0.7
    container_name: mysql-bkup
    command:
      - /bin/sh
      - -c
      - mysql-bkup restore --file database_20231217_115621.sql -d database_name
    volumes:
      - ./backup:/backup
    environment:
      #- FILE_NAME=database_20231217_040238.sql.gz # Optional if file name is set from command
      - DB_PORT=3306
      - DB_HOST=postgres
      - DB_USERNAME=user_name
      - DB_PASSWORD=password
```
## Run

```sh
docker-compose up -d
```
## Backup to S3

```sh
docker run --rm --privileged \
--device /dev/fuse --name mysql-bkup \
-e "DB_HOST=db_hostname" \
-e "DB_USERNAME=username" \
-e "DB_PASSWORD=password" \
-e "ACCESS_KEY=your_access_key" \
-e "SECRET_KEY=your_secret_key" \
-e "BUCKETNAME=your_bucket_name" \
-e "S3_ENDPOINT=https://s3.us-west-2.amazonaws.com" \
jkaninda/mysql-bkup:v0.7  mysql-bkup backup -s s3 -d database_name
```
> To change s3 backup path add this flag : --path /my_customPath . default path is /mysql-bkup

Simple S3 backup usage

```sh
mysql-bkup backup --storage s3 --dbname mydatabase 
```
```yaml
  mysql-bkup:
    image: jkaninda/mysql-bkup:v0.7
    container_name: mysql-bkup
    privileged: true
    devices:
    - "/dev/fuse"
    command:
      - /bin/sh
      - -c
      - mysql-bkup restore --storage s3 -f database_20231217_115621.sql.gz --dbname database_name
    environment:
      - DB_PORT=3306
      - DB_HOST=postgress
      - DB_USERNAME=user_name
      - DB_PASSWORD=password
      - ACCESS_KEY=${ACCESS_KEY}
      - SECRET_KEY=${SECRET_KEY}
      - BUCKET_NAME=${BUCKET_NAME}
      - S3_ENDPOINT=${S3_ENDPOINT}

```
## Run in Scheduled mode

This tool can be run as CronJob in Kubernetes for a regular backup which makes deployment on Kubernetes easy as Kubernetes has CronJob resources.
For Docker, you need to run it in scheduled mode by adding `--mode scheduled` flag and specify the periodical backup time by adding `--period "0 1 * * *"` flag.

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

> At every 30th minute

```conf
*/30 * * * *
```
> “At minute 0.” every hour
```conf
0 * * * *
```

> “At 01:00.” every day

```conf
0 1 * * *
```

## Example of scheduled mode

> Docker run :

```sh
docker run --rm --name mysql-bkup \
-v $BACKUP_DIR:/backup/ \
-e "DB_HOST=$DB_HOST" \
-e "DB_USERNAME=$DB_USERNAME" \
-e "DB_PASSWORD=$DB_PASSWORD" jkaninda/mysql-bkup:v0.7  mysql-bkup backup --dbname $DB_NAME --mode scheduled --period "0 1 * * *"
```

> With Docker compose

```yaml
version: "3"
services:
  mysql-bkup:
    image: jkaninda/mysql-bkup:v0.7
    container_name: mysql-bkup
    privileged: true
    devices:
    - "/dev/fuse"
    command:
      - /bin/sh
      - -c
      - mysql-bkup backup --storage s3 --path /mys3_custom_path --dbname database_name --mode scheduled --period "*/30 * * * *"
    environment:
      - DB_PORT=3306
      - DB_HOST=postgreshost
      - DB_USERNAME=userName
      - DB_PASSWORD=${DB_PASSWORD}
      - ACCESS_KEY=${ACCESS_KEY}
      - SECRET_KEY=${SECRET_KEY}
      - BUCKET_NAME=${BUCKET_NAME}
      - S3_ENDPOINT=${S3_ENDPOINT}
```

## Kubernetes CronJob

For Kubernetes, you don't need to run it in scheduled mode.

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
            image: jkaninda/mysql-bkup:v0.7
            securityContext:
              privileged: true
            command:
            - /bin/sh
            - -c
            - mysql-bkup backup -s s3 --path /custom_path
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
              - name: ACCESS_KEY
                value: ""
              - name: SECRET_KEY
                value: ""
              - name: BUCKET_NAME
                value: ""
              - name: S3_ENDPOINT
                value: "https://s3.us-west-2.amazonaws.com"
          restartPolicy: Never
```

## Authors

**Jonas Kaninda**
- <https://github.com/jkaninda>

