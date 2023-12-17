FROM ubuntu:22.04
ENV DB_HOST=""
ENV DB_DATABASE=""
ENV DB_USERNAME=""
ENV DB_PASSWORD=""
ENV DB_PORT="3306"
ENV STORAGE=local
ENV BUCKETNAME=""
ENV ACCESS_KEY=""
ENV SECRET_KEY=""
ENV S3_ENDPOINT=https://s3.amazonaws.com
ARG DEBIAN_FRONTEND=noninteractive

RUN apt-get update -qq
RUN apt-get install build-essential libcurl4-openssl-dev libxml2-dev mime-support -y
RUN apt install s3fs mysql-client -y
RUN mkdir /s3mnt
RUN mkdir /tmp/s3cache
RUN chmod 777 /s3mnt
RUN chmod 777 /tmp/s3cache

COPY src/backup.sh /usr/local/bin/
COPY src/restore.sh /usr/local/bin/
RUN chmod +x /usr/local/bin/backup.sh
RUN chmod +x /usr/local/bin/restore.sh

RUN ln -s /usr/local/bin/backup.sh /usr/local/bin/backup
RUN ln -s /usr/local/bin/restore.sh /usr/local/bin/restore

RUN mkdir /backup
WORKDIR /backup