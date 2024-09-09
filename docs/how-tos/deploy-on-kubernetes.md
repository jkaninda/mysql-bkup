---
title: Deploy on Kubernetes
layout: default
parent: How Tos
nav_order: 8
---

## Deploy on Kubernetes

To deploy MySQL Backup on Kubernetes, you can use Job to backup or Restore your database.
For recurring backup you can use CronJob, you don't need to run it in scheduled mode. as described bellow.

## Backup to S3 storage

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: backup
spec:
  template:
    spec:
      containers:
        - name: mysql-bkup
          # In production, it is advised to lock your image tag to a proper
          # release version instead of using `latest`.
          # Check https://github.com/jkaninda/mysql-bkup/releases
          # for a list of available releases.
          image: jkaninda/mysql-bkup
          command:
            - /bin/sh
            - -c
            - backup --storage s3
          resources:
            limits:
              memory: "128Mi"
              cpu: "500m"
          env:
            - name: DB_PORT
              value: "3306"
            - name: DB_HOST
              value: ""
            - name: DB_NAME
              value: "dbname"
            - name: DB_USERNAME
              value: "username"
            # Please use secret!
            - name: DB_PASSWORD
              value: ""
            - name: AWS_S3_ENDPOINT
              value: "https://s3.amazonaws.com"
            - name: AWS_S3_BUCKET_NAME
              value: "xxx"
            - name: AWS_REGION
              value: "us-west-2"
            - name: AWS_ACCESS_KEY
              value: "xxxx"
            - name: AWS_SECRET_KEY
              value: "xxxx"
            - name: AWS_DISABLE_SSL
              value: "false"
      restartPolicy: Never
```

## Backup Job to SSH remote server

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: backup
spec:
  ttlSecondsAfterFinished: 100
  template:
    spec:
      containers:
      - name: mysql-bkup
        # In production, it is advised to lock your image tag to a proper
        # release version instead of using `latest`.
        # Check https://github.com/jkaninda/mysql-bkup/releases
        # for a list of available releases.
        image: jkaninda/mysql-bkup
        command:
        - /bin/sh
        - -c
        - bkup
        - backup
        - --storage
        - ssh
        - --disable-compression
        resources:
          limits:
            memory: "128Mi"
            cpu: "500m"
        env:
          - name: DB_PORT
            value: "3306"
          - name: DB_HOST
            value: ""
          - name: DB_NAME
            value: "dbname"
          - name: DB_USERNAME
            value: "username"
          # Please use secret!
          - name: DB_PASSWORD
            value: ""
          - name: SSH_HOST_NAME
            value: "xxx"
          - name: SSH_PORT
            value: "22"
          - name: SSH_USER
            value: "xxx"
          - name: SSH_PASSWORD
            value: "xxxx"
          - name: SSH_REMOTE_PATH
            value: "/home/toto/backup"
          # Optional, required if you want to encrypt your backup
          - name: GPG_PASSPHRASE
            value: "xxxx"
      restartPolicy: Never
```

## Restore Job

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: restore-job
spec:
  ttlSecondsAfterFinished: 100
  template:
    spec:
      containers:
      - name: mysql-bkup
        # In production, it is advised to lock your image tag to a proper
        # release version instead of using `latest`.
        # Check https://github.com/jkaninda/mysql-bkup/releases
        # for a list of available releases.
        image: jkaninda/mysql-bkup
        command:
        - /bin/sh
        - -c
        - bkup
        - restore
        - --storage
        - ssh
        - --file store_20231219_022941.sql.gz
        resources:
          limits:
            memory: "128Mi"
            cpu: "500m"
        env:
        - name: DB_PORT
          value: "3306" 
        - name: DB_HOST
          value: ""
        - name: DB_NAME
          value: "dbname"
        - name: DB_USERNAME
          value: "username"
        # Please use secret!
        - name: DB_PASSWORD
          value: ""
        - name: SSH_HOST_NAME
          value: "xxx"
        - name: SSH_PORT
          value: "22"
        - name: SSH_USER
          value: "xxx"
        - name: SSH_PASSWORD
          value: "xxxx"
        - name: SSH_REMOTE_PATH
          value: "/home/xxxx/backup"
          # Optional, required if your backup was encrypted
        #- name: GPG_PASSPHRASE
        #  value: "xxxx"
      restartPolicy: Never
```

## Recurring backup

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: backup-job
spec:
  schedule: "* * * * *"
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: mysql-bkup
            image: jkaninda/mysql-bkup
            command:
            - /bin/sh
            - -c
            - bkup
            - backup
            - --storage
            - ssh
            - --disable-compression
            resources:
              limits:
                memory: "128Mi"
                cpu: "500m"
            env:
            - name: DB_PORT
              value: "3306" 
            - name: DB_HOST
              value: ""
            - name: DB_NAME
              value: "username"
            - name: DB_USERNAME
              value: "username"
            # Please use secret!
            - name: DB_PASSWORD
              value: ""
            - name: SSH_HOST_NAME
              value: "xxx"
            - name: SSH_PORT
              value: "xxx"
            - name: SSH_USER
              value: "jkaninda"
            - name: SSH_REMOTE_PATH
              value: "/home/jkaninda/backup"
            - name: SSH_PASSWORD
              value: "password"
            # Optional, required if you want to encrypt your backup
            #- name: GPG_PASSPHRASE
            #  value: "xxx"
          restartPolicy: Never
```

## Kubernetes Rootless

This image also supports Kubernetes security context, you can run it in Rootless environment.
It has been tested on Openshift, it works well.
Deployment on OpenShift is supported, you need to remove `securityContext` section on your yaml file.

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: backup-job
spec:
  schedule: "* * * * *"
  jobTemplate:
    spec:
      template:
        spec:
          securityContext:
            runAsUser: 1000
            runAsGroup: 3000
            fsGroup: 2000
          containers:
            # In production, it is advised to lock your image tag to a proper
            # release version instead of using `latest`.
            # Check https://github.com/jkaninda/mysql-bkup/releases
            # for a list of available releases.
            - name: mysql-bkup
              image: jkaninda/mysql-bkup
              command:
                - /bin/sh
                - -c
                - bkup
                - backup
                - --storage
                - ssh
                - --disable-compression
              resources:
                limits:
                  memory: "128Mi"
                  cpu: "500m"
              env:
                - name: DB_PORT
                  value: "3306"
                - name: DB_HOST
                  value: ""
                - name: DB_NAME
                  value: "xxx"
                - name: DB_USERNAME
                  value: "xxx"
                # Please use secret!
                - name: DB_PASSWORD
                  value: ""
                - name: SSH_HOST_NAME
                  value: "xxx"
                - name: SSH_PORT
                  value: "22"
                - name: SSH_USER
                  value: "jkaninda"
                - name: SSH_REMOTE_PATH
                  value: "/home/jkaninda/backup"
                - name: SSH_PASSWORD
                  value: "password"
              # Optional, required if you want to encrypt your backup
              #- name: GPG_PASSPHRASE
              #  value: "xxx"
          restartPolicy: OnFailure
```
