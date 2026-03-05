#!/bin/bash

PIPE_PATH=/srv/minecraft/mginx/scripts/pipe
COMPOSE_PATH=/home/opisek/docker/minecraft/docker-compose.yml

if [[ ! -p $PIPE_PATH ]]; then
  mkfifo $PIPE_PATH
fi

while true; do
  CMD=`cat ${PIPE_PATH}`
  PARTS=($CMD)

  if [[ "${#PARTS[@]}" -ne 2 ]]; then
    continue
  fi

  case ${PARTS[0]} in
    start)
      docker compose -f $COMPOSE_PATH up -d ${PARTS[1]}
    ;;
    stop)
      docker compose -f $COMPOSE_PATH stop ${PARTS[1]}
    ;;
  esac
done
