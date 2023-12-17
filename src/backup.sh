#!/bin/sh
TIME=$(date +%Y%m%d_%H%M%S)
MY_SQL_DUMP=/usr/bin/mysqldump
#OPTION=${OPTION}
set -e
if [ -z "${DB_HOST}"] ||  [ -z "${DB_DATABASE}"] ||  [ -z "${DB_USERNAME}"] ||  [ -z "${DB_PASSWORD}"]; then
   echo "Please make sure all environment variables are set "
else
      ## Backup database
      mysqldump -h ${DB_HOST} -P ${DB_PORT} -u ${DB_USERNAME} --password=${DB_PASSWORD} ${DB_DATABASE} > /backup/${DB_DATABASE}_${TIME}.sql
      echo "Database has been saved"
fi
exit