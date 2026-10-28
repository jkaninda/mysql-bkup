# syntax=docker/dockerfile:1
ARG GO_IMAGE=golang:1.26.3-trixie
ARG RUNTIME_IMAGE=debian:13.7-slim
ARG MYSQL_IMAGE=mysql:8.4.13

FROM --platform=$BUILDPLATFORM ${GO_IMAGE} AS build
WORKDIR /app
ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT
ARG appVersion="development"
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN GOARM="${TARGETVARIANT#v}" CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags="-s -w -X main.Version=${appVersion}" -o /out/mysql-bkup .

# Oracle MySQL client binaries, per architecture.
FROM ${MYSQL_IMAGE} AS mysqlclient-amd64
RUN mkdir -p /out && cp /usr/bin/mysql /usr/bin/mysqldump /out/
FROM ${MYSQL_IMAGE} AS mysqlclient-arm64
RUN mkdir -p /out && cp /usr/bin/mysql /usr/bin/mysqldump /out/
FROM ${RUNTIME_IMAGE} AS mysqlclient-arm
RUN mkdir -p /out
FROM mysqlclient-${TARGETARCH} AS mysqlclient

FROM ${RUNTIME_IMAGE}
ARG appVersion="development"
ENV TZ=UTC
LABEL org.opencontainers.image.title="mysql-bkup" \
      org.opencontainers.image.description="Back up and restore MySQL and MariaDB databases to local storage, S3, GCS, Azure, SFTP or FTP, with GPG or age encryption" \
      org.opencontainers.image.licenses="MIT" \
      org.opencontainers.image.authors="Jonas Kaninda <me@jkaninda.dev>" \
      org.opencontainers.image.version=${appVersion} \
      org.opencontainers.image.source="https://github.com/jkaninda/mysql-bkup"


RUN apt-get update \
 && apt-get install -y --no-install-recommends \
      mariadb-client-core libssl3t64 zlib1g libncurses6 libtinfo6 \
      tini tzdata ca-certificates \
 && cd /tmp && apt-get download mariadb-client \
 && dpkg-deb --fsys-tarfile mariadb-client_*.deb | tar -x -C / ./usr/bin/mariadb-dump \
 && rm -f mariadb-client_*.deb \
 && rm -rf /var/lib/apt/lists/*

# /usr/local/bin comes first in PATH: mysql and mysqldump are Oracle's there,
# while Debian's /usr/bin/mysql* are links to the MariaDB tools.
COPY --from=mysqlclient /out/ /usr/local/bin/
COPY --from=build /out/mysql-bkup /usr/local/bin/mysql-bkup


RUN mkdir -p /config /backup /tmp/backup \
 && chmod 1777 /backup /tmp/backup \
 && ln -s /usr/local/bin/mysql-bkup /usr/local/bin/bkup \
 && printf '#!/bin/sh\nexec /usr/local/bin/mysql-bkup backup "$@"\n' > /usr/local/bin/backup \
 && printf '#!/bin/sh\nexec /usr/local/bin/mysql-bkup restore "$@"\n' > /usr/local/bin/restore \
 && printf '#!/bin/sh\nexec /usr/local/bin/mysql-bkup migrate "$@"\n' > /usr/local/bin/migrate \
 && chmod +x /usr/local/bin/backup /usr/local/bin/restore /usr/local/bin/migrate \
 && (mysqldump --version 2>/dev/null || true) && mariadb-dump --version && mysql-bkup version

WORKDIR /config
HEALTHCHECK --interval=1m --timeout=10s --start-period=1m CMD ["/usr/local/bin/mysql-bkup", "health"]
ENTRYPOINT ["/usr/bin/tini", "--", "/usr/local/bin/mysql-bkup"]
