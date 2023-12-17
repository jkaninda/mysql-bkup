#!/bin/sh 
set -e
TIME=$(date +%Y%m%d_%H%M%S)
MY_SQL_DUMP=/usr/bin/mysqldump
arg0=$(basename "$0" .sh)
blnk=$(echo "$arg0" | sed 's/./ /g')
export OPERATION=backup
export DESTINATION=local
export DESTINATION_DIR=/backup
export SOURCE=local
usage_info()
{
    echo "Usage: \\"
    echo "     $blnk Backup: mysql_bkup -o backup -d s3 \\"
    echo "     $blnk Restore: mysql_bkup -o restore -s s3 -f my_db.sql \\"
    echo "     $blnk [-o|--operation] [{-d|--destination} ] [{-f|--file} ] [{-s|--source} ] [{-h|--help} ] \\"

}
version_info()
{
   echo "Version: 1.0"
   exit 0
}
usage()
{
    exec 1>2   # Send standard output to standard error
    usage_info
    exit 0
}

error()
{
    echo "$arg0: $*" >&2
    exit 0
}

help()
{
    echo
    echo "  -o|--operation         -- Set operation (default: backup)"
    echo "  -d|--destination       -- Set destination (default: local)"
    echo "  -s|--source            -- Set source (default: local)"
    echo "  -s|--file              -- Set file name "
    echo "  -t|--timeout           -- Set timeout (default: 120s)"
    echo "  -h|--help              -- Print this help message and exit"
    echo "  -v|--version           -- Print version information and exit"
    exit 0
}

flags()
{
    while test $# -gt 0
    do
        case "$1" in
        (-o|--operation)
            shift
            [ $# = 0 ] && error "No operation specified - restore or backup"
            export OPERATION="$1"
            shift;;
        (-d|--destination)
            shift
            [ $# = 0 ] && error "No destination specified - local or s3 | default local"
            export DESTINATION="$1"
            export SOURCE="$1"
            shift;;
        (-s|--source)
            shift
            [ $# = 0 ] && error "No source specified - local or s3 | default local"
            export SOURCE="$1"
            export DESTINATION="$1"
            shift;;
        (-f|--file)
            shift
            [ $# = 0 ] && error "No file specified - file to restore"
            export FILE_NAME="$1"
            shift;;
        (-t|--timeout)
            shift
            [ $# = 0 ] && error "No timeout specified"
            export TIMEOUT="$1"
            shift;;   
        (-h|--help)
            help;;
        (-V|--version)
           version_info;;
        (--)
           help;;
        (*) usage;;
        esac
    done
}

backup()
{
 if [ -z "${DB_HOST}"] ||  [ -z "${DB_DATABASE}"] ||  [ -z "${DB_USERNAME}"] ||  [ -z "${DB_PASSWORD}"]; then
   echo "Please make sure all required options are set "
else
      ## Backup database
      mysqldump -h ${DB_HOST} -P ${DB_PORT} -u ${DB_USERNAME} --password=${DB_PASSWORD} ${DB_DATABASE} > ${DESTINATION_DIR}/${DB_DATABASE}_${TIME}.sql
      echo "Database has been saved"
fi
exit
}

restore()
{
if [ -z "${DB_HOST}" ] ||  [ -z "${DB_DATABASE}" ] ||  [ -z "${DB_USERNAME}" ] || [ -z "${DB_PASSWORD}" ]; then
   echo "Please make sure all required options are set "
else
    ## Restore database
     if [ -f "${DESTINATION_DIR}/$FILE_NAME" ]; then
       cat ${DESTINATION_DIR}/${FILE_NAME} | mysql -h ${DB_HOST} -P ${DB_PORT} -u ${DB_USERNAME} --password=${DB_PASSWORD} ${DB_DATABASE}
        echo "Database has been restored"
      else
        echo "Error, file not found in /backup folder"
      fi 
fi
exit
}

s3_backup()
{
 echo "S3 Backup"
 mount_s3
 backup
}

s3_restore()
{
  echo "S3 Restore"
   mount_s3
   restore

}

mount_s3()
{
if [ -z "${ACCESS_KEY}"] ||  [ -z "${SECRET_KEY}"]; then
echo "Please make sure all environment variables are set "
echo "BUCKETNAME=$BUCKETNAME \nACCESS_KEY=$nACCESS_KEY \nSECRET_KEY=$SECRET_KEY"
else
echo "$ACCESS_KEY:$SECRET_KEY" | tee /etc/passwd-s3fs
chmod 600 /etc/passwd-s3fs
echo "Mounting Object storage in /s3mnt .... "
s3fs $BUCKETNAME /s3mnt -o passwd_file=/etc/passwd-s3fs -o use_cache=/tmp/s3cache -o allow_other -o url=$S3_ENDPOINT -o use_path_request_style
ls /s3mnt | wc -l
export DESTINATION_DIR=/s3mnt
fi
}
flags "$@"
# ?
  if [  $OPERATION != 'backup' ]
  then
     if [ $DESTINATION != 's3' ]
     then
          echo "Restore from local"
          restore
      else
        echo "Restore from s3"
        s3_restore
      fi
  else
      if [ $DESTINATION != 's3' ]
      then
          echo "Backup to local destination"
          backup
      else
         echo "Restore from s3"
         s3_backup
      fi
   fi