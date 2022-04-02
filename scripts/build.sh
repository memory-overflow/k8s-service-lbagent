#!/usr/bin/env bash

set -e

WK_DIR=$(
    cd $(dirname ${BASH_SOURCE[0]})
    pwd
)/../

cd ${WK_DIR}

go version

base_dir=$(pwd)

rm -rf pack
rm -rf pack.tgz

go mod tidy

go build -o pack/bin/media_manage cmd/media_manage/main.go
go build -o pack/bin/task_manage cmd/task_manage/main.go
go build -o pack/bin/toolkit_manage cmd/toolkit_manage/main.go
go build -o pack/bin/task_schedule cmd/task_schedule/main.go
go build -o pack/bin/register_api cmd/tools/register_api/main.go

cp -r "${base_dir}"/conf pack
cp -r "${base_dir}"/scripts pack

ls -Ral pack*
