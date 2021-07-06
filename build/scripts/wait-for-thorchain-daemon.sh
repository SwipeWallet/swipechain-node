#!/bin/sh

# https://docs.docker.com/compose/startup-order/

set -e

echo "Waiting for THORChain RPC..."

until curl -s "$1" >/dev/null; do
  echo "THORChain RPC is unavailable - sleeping ($1)"
  sleep 3
done

sleep 5 # wait for first block to become available

echo "THORChain RPC ready"
