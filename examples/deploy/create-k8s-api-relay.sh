#!/bin/bash

SERVER_URL="$1"
SERVICE_NAME="$2"
SERVICE_TOKEN="$3"
TARGET_ADDR="$4"

if [ -z "$SERVER_URL" ] || [ -z "$SERVICE_NAME" ] || [ -z "$SERVICE_TOKEN" ] || [ -z "$TARGET_ADDR" ]; then
  echo "Usage: $0 <SERVER_URL> <SERVICE_NAME> <SERVICE_TOKEN> <TARGET_ADDR>"
  echo ""
  echo "Arguments:"
  echo "  SERVER_URL      WebSocket tunnel server URL"
  echo "  SERVICE_NAME    Name of the service to register"
  echo "  SERVICE_TOKEN   Authentication token for the service"
  echo "  TARGET_ADDR     Target address to forward traffic to"
  exit 1
fi

set -eu

cd "$(dirname "$0")"

jsonnet -V "SERVER_URL=$SERVER_URL" -V "SERVICE_NAME=$SERVICE_NAME" -V "SERVICE_TOKEN=$SERVICE_TOKEN" -V "TARGET_ADDR=$TARGET_ADDR" create-k8s-api-relay.jsonnet | yq -P -o yaml '.[] | split_doc'