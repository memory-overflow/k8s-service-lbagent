#!/bin/sh

set -e
# set -o pipefail

WK_DIR=$(cd $(dirname ${BASH_SOURCE[0]}); pwd )/../

cd ${WK_DIR}

number=$1

version=`git log  | grep commit | awk {'print $2'} | head -n 1`
dockername="jisuanke/k8s-service-lbagent:v${number}-${version}"

dockername="jisuanke/k8s-service-lbagent:latest"

if [[ "${number}" == "debug" ]]; then
  go mod vendor
  go mod tidy
  echo "Start build ${dockername}"
  docker build -t ${dockername} ./ -f./Dockerfile_debug
  echo "Built ${dockername} success"
  rm -rf vendor
else
  sh scripts/build.sh
  echo "Start build ${dockername}"
  echo "docker build -t ${dockername} ."
  docker build -t ${dockername} .
  echo "Built ${dockername} success"
fi

echo "Start push ${dockername}"
docker push ${dockername}
echo "Pushed ${dockername}"