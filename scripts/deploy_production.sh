#!/bin/bash

set -e
set -o pipefail
export LANG=en_US.UTF-8

start_server() {
  WK_DIR=/usr/local/services/ai-media
  cd $WK_DIR
  mkdir -p ${ENV_LOG_DIR} || true
  ln -sf ${ENV_LOG_DIR} logs
  # 创建数据库
  sleep 7
  echo "CREATE DATABASE IF NOT EXISTS $ENV_MYSQL_DB_NAME" | mysql -h$DB_HOST -u$DB_USER -p$DB_PASSWORD -P$DB_PORT

  ./bin/${APP_NAME} -conf=./conf/trpc/${APP_NAME}.yaml
}

if [ "$1" == "" ]; then
  echo "[Usage] sh $0.sh [start|stop|restart]"
  exit
fi

if [ "$1" == "restart" ]; then
  pkill -9 ${APP_NAME}
  sleep 1
  start_server
elif [ "$1" == "stop" ]; then
  pkill -9 ${APP_NAME}
  sleep 1
elif [ "$1" == "start" ]; then
  start_server
else
  echo "[Usage] sh $0.sh [start|stop|restart]"
fi