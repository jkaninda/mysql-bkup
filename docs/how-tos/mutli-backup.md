---
title: Run multiple backup schedules in the same container
layout: default
parent: How Tos
nav_order: 11
---

Multiple backup schedules with different configuration can be configured by mounting a configuration file into `/config/config.yaml`  `/config/config.yml` or by defining an environment variable `BACKUP_CONFIG_FILE=/backup/config.yaml`.

## Configuration file

```yaml
#cronExpression: "@every 20m" //Optional for scheduled backups
cronExpression: "" 
databases:
  - host: mysql1
    port: 3306
    name: database1
    user: database1
    password: password
    path: /s3-path/database1 #For SSH or FTP you need to define the full path (/home/toto/backup/)
  - host: mysql2
    port: 3306
    name: lldap
    user: lldap
    password: password
    path: /s3-path/lldap #For SSH or FTP you need to define the full path (/home/toto/backup/)
  - host: mysql3
    port: 3306
    name: keycloak
    user: keycloak
    password: password
    path: /s3-path/keycloak #For SSH or FTP you need to define the full path (/home/toto/backup/)
  - host: mysql4
    port: 3306
    name: joplin
    user: joplin
    password: password
    path: /s3-path/joplin #For SSH or FTP you need to define the full path (/home/toto/backup/)
```
## Docker compose file

```yaml
services:
  mysql-bkup:
    # In production, it is advised to lock your image tag to a proper
    # release version instead of using `latest`.
    # Check https://github.com/jkaninda/mysql-bkup/releases
    # for a list of available releases.
    image: jkaninda/mysql-bkup
    container_name: mysql-bkup
    command: backup
    volumes:
      - ./backup:/backup
    environment:
      ## Multi backup config file
      - BACKUP_CONFIG_FILE=/backup/config.yaml
    # mysql-bkup container must be connected to the same network with your database
    networks:
      - web
networks:
  web:
```