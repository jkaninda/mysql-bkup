---
title: Migrate database
layout: default
parent: How Tos
nav_order: 10
---

# Migrate Database

To migrate a MySQL database from a source to a target database, you can use the `migrate` command. This feature simplifies the process by combining the backup and restore operations into a single step.

{: .note }
The `migrate` command eliminates the need for separate backup and restore operations. It directly transfers data from the source database to the target database.

{: .warning }
The `migrate` operation is **irreversible**. Always back up your target database before performing this action.

---

## Configuration Steps

1. **Source Database**: Provide connection details for the source database.
2. **Target Database**: Provide connection details for the target database.
3. **Run the Migration**: Use the `migrate` command to initiate the migration.

---

## Example: Docker Compose Configuration

Below is an example `docker-compose.yml` configuration for migrating a database:

```yaml
services:
  mysql-bkup:
    # In production, lock your image tag to a specific release version
    # instead of using `latest`. Check https://github.com/jkaninda/mysqlbkup/releases
    # for available releases.
    image: jkaninda/mysql-bkup
    container_name: mysql-bkup
    command: migrate
    volumes:
      - ./backup:/backup
    environment:
      ## Source Database
      - DB_PORT=3306
      - DB_HOST=mysql
      - DB_NAME=database
      - DB_USERNAME=username
      - DB_PASSWORD=password

      ## Target Database
      - TARGET_DB_HOST=target-postgres
      - TARGET_DB_PORT=3306
      - TARGET_DB_NAME=dbname
      - TARGET_DB_USERNAME=username
      - TARGET_DB_PASSWORD=password

    # Ensure the mysql-bkup container is connected to the same network as your database
    networks:
      - web

networks:
  web:
```

---

## Migrate Database Using Docker CLI

You can also run the migration directly using the Docker CLI. Below is an example:

### Environment Variables

Save your source and target database connection details in an environment file (e.g., `your-env`):

```bash
## Source Database
DB_HOST=postgres
DB_PORT=3306
DB_NAME=dbname
DB_USERNAME=username
DB_PASSWORD=password

## Target Database
TARGET_DB_HOST=target-postgres
TARGET_DB_PORT=3306
TARGET_DB_NAME=dbname
TARGET_DB_USERNAME=username
TARGET_DB_PASSWORD=password
```

### Run the Migration

```bash
docker run --rm --network your_network_name \
  --env-file your-env \
  -v $PWD/backup:/backup/ \
  jkaninda/pg-bkup migrate
```

---

## Key Notes

- **Irreversible Operation**: The `migrate` command directly transfers data from the source to the target database. Ensure you have a backup of the target database before proceeding.
- **Network Configuration**: Ensure the `mysql-bkup` container is connected to the same network as your source and target databases.
