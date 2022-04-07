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

go build -o pack/bin/agent cmd/main.go

cp -r "${base_dir}"/conf pack

ls -Ral pack*
