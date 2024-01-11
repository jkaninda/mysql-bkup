#!/usr/bin/env bash
if [ $# -eq 0 ]
  then
    tag='latest'
  else
    tag=$1
fi

docker build -f src/Dockerfile  -t jkaninda/mysql-bkup:$tag .

#docker compose up -d