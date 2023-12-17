#!/bin/sh
TIME=$(date +%Y%m%d_%H%M%S)
MY_SQL_DUMP=/usr/bin/mysqldump
set -e

if [ -z "${DB_HOST}"] ||  [ -z "${DB_DATABASE}"] ||  [ -z "${DB_USERNAME}"] ||  [ -z "${DB_PASSWORD}"]; then
   echo "Please make sure all environment variables are set "
else
  if [  $OPTION != 'backup' ]
  then
     ## Restore databas
     echo "Restoring database..."
     if [ -f "/backup/$FILE_NAME" ]; then
       cat /backup/${FILE_NAME} | mysql -h ${DB_HOST} -P ${DB_PORT} -u ${DB_USERNAME} --password=${DB_PASSWORD} ${DB_DATABASE}
        echo "Database has been restored"

      else
        echo "Error, file not found in /backup folder"
      fi 
  else
      ## Backup database
      echo "Start backup database..."
      mysqldump -h ${DB_HOST} -P ${DB_PORT} -u ${DB_USERNAME} --password=${DB_PASSWORD} ${DB_DATABASE} > /backup/${DB_DATABASE}_${TIME}.sql
      echo "Database has been saved"

       
  fi
fi
bash