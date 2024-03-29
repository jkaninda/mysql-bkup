FROM golang:1.21.0 AS build
WORKDIR /app

# Copy the source code.
COPY . .
# Installs Go dependencies
RUN go mod download

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/mysql-bkup

FROM ubuntu:24.04
ENV DB_HOST=""
ENV DB_NAME=""
ENV DB_USERNAME=""
ENV DB_PASSWORD=""
ENV DB_PORT="3306"
ENV STORAGE=local
ENV BUCKET_NAME=""
ENV ACCESS_KEY=""
ENV SECRET_KEY=""
ENV S3_ENDPOINT=https://s3.amazonaws.com
ARG DEBIAN_FRONTEND=noninteractive
ENV VERSION="v0.7"
LABEL authors="Jonas Kaninda"

RUN apt-get update -qq
#RUN apt-get install build-essential libcurl4-openssl-dev libxml2-dev mime-support -y
RUN apt install s3fs mysql-client supervisor cron -y

# Clear cache
RUN apt-get clean && rm -rf /var/lib/apt/lists/*

RUN mkdir /s3mnt
RUN mkdir /tmp/s3cache
RUN chmod 777 /s3mnt
RUN chmod 777 /tmp/s3cache

COPY --from=build /app/mysql-bkup /usr/local/bin/mysql-bkup
RUN chmod +x /usr/local/bin/mysql-bkup

RUN ln -s /usr/local/bin/mysql-bkup /usr/local/bin/bkup
RUN ln -s /usr/local/bin/mysql-bkup /usr/local/bin/mysql_bkup


ADD docker/supervisord.conf /etc/supervisor/supervisord.conf


RUN mkdir /backup
WORKDIR /backup