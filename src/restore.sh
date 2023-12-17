#!/bin/sh
TIME=$(date +%Y%m%d_%H%M%S)
MY_SQL_DUMP=/usr/bin/mysqldump
set -e
if [ -z "${DB_HOST}"] ||  [ -z "${DB_DATABASE}"] ||  [ -z "${DB_USERNAME}"] ||  [ -z "${DB_PASSWORD}"]; then
   echo "Please make sure all environment variables are set "
else
    ## Restore database
     if [ -f "/backup/$FILE_NAME" ]; then
       cat /backup/${FILE_NAME} | mysql -h ${DB_HOST} -P ${DB_PORT} -u ${DB_USERNAME} --password=${DB_PASSWORD} ${DB_DATABASE}
        echo "Database has been restored"
      else
        echo "Error, file not found in /backup folder"
      fi 
fi
exit