#!/bin/sh

set -o pipefail

. "$(dirname "$0")/core.sh"

SEEDS="${SEEDS:=none}" # the hostname of multiple seeds set as tendermint seeds
if [ ! -f ~/.thornode/config/genesis.json ]; then
  if [ "$PEER" = "none" ] && [ "$SEEDS" = "none" ]; then
    echo "Missing PEER / SEEDS"
    exit 1
  fi

  init_chain

  if [ "$SEEDS" != "none" ]; then
    fetch_genesis_from_seeds $SEEDS

    # add seeds tendermint config
    seeds_list $SEEDS
  fi

  # enable telemetry through prometheus metrics endpoint
  enable_telemetry

  # enable internal traffic as well
  enable_internal_traffic

  # use external IP if available
  [ -n "$EXTERNAL_IP" ] && external_address "$EXTERNAL_IP" "$NET"

else
  if [ "$SEEDS" != "none" ]; then
    # add seeds tendermint config
    seeds_list $SEEDS
  fi
fi

exec "$@"
