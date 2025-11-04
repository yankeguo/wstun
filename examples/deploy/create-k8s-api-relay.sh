#!/bin/bash

set -eu

cd "$(dirname "$0")"

jsonnet -V "SERVER_URL=$1" -V "SERVICE_NAME=$2" -V "SERVICE_TOKEN=$3" -V "TARGET_ADDR=$4" create-k8s-api-relay.jsonnet | yq -P -o yaml '.[] | split_doc'