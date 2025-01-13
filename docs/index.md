---
title: Overview
layout: home
nav_order: 1
---

# About mysql-bkup
{:.no_toc}

**MYSQL-BKUP** is a Docker container image designed to **backup, restore, and migrate MySQL databases**.
It supports a variety of storage options and ensures data security through GPG encryption.

---

## Key Features

### Storage Options
- **Local storage**
- **AWS S3** or any S3-compatible object storage
- **FTP**
- **SFTP**
- **SSH-compatible storage**
- **Azure Blob storage**

### Data Security
- Backups can be encrypted using **GPG** to ensure data confidentiality.

### Deployment Flexibility
- Available as the [jkaninda/pg-bkup](https://hub.docker.com/r/jkaninda/mysql-bkup) Docker image.
- Deployable on **Docker**, **Docker Swarm**, and **Kubernetes**.
- Supports recurring backups of MySQL databases:
    - On Docker for automated backup schedules.
    - As a **Job** or **CronJob** on Kubernetes.

### Notifications
- Receive real-time updates on backup success or failure via:
    - **Telegram**
    - **Email**

---

## Use Cases

- **Automated Recurring Backups:** Schedule regular backups for MySQL databases.
- **Cross-Environment Migration:** Easily migrate MySQL databases across different environments using supported storage options.
- **Secure Backup Management:** Protect your data with GPG encryption.

---

## Get Involved

We welcome contributions! Feel free to give us a ⭐, submit PRs, or open issues on our [GitHub repository](https://github.com/jkaninda/mysql-bkup).

{: .fs-6 .fw-300 }

---

{: .note }
Code and documentation for the `v1` version are available on [this branch][v1-branch].

[v1-branch]: https://github.com/jkaninda/mysql-bkup

---

## Available Image Registries

The Docker image is published to both **Docker Hub** and the **GitHub Container Registry**. You can use either of the following:

```bash
docker pull jkaninda/mysql-bkup
docker pull ghcr.io/jkaninda/mysql-bkup
```

While the documentation references Docker Hub, all examples work seamlessly with `ghcr.io`.

---

## References

We created this image as a simpler and more lightweight alternative to existing solutions. Here’s why:

- **Lightweight:** Written in Go, the image is optimized for performance and minimal resource usage.
- **Multi-Architecture Support:** Supports `arm64` and `arm/v7` architectures.
- **Docker Swarm Support:** Fully compatible with Docker in Swarm mode.
- **Kubernetes Support:** Designed to work seamlessly with Kubernetes.
