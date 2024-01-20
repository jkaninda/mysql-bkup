#!/bin/sh
DB_USERNAME='db_username'
DB_PASSWORD='password'
DB_HOST='db_hostname'
DB_NAME='db_name'
BACKUP_DIR="$PWD/backup"

docker run --rm --name mysql-bkup -v $BACKUP_DIR:/backup/ -e "DB_HOST=$DB_HOST" -e "DB_USERNAME=$DB_USERNAME" -e "DB_PASSWORD=$DB_PASSWORD" jkaninda/mysql-bkup:latest backup -d $DB_NAME