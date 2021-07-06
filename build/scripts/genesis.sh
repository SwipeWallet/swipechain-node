#!/bin/sh

set -o pipefail

. "$(dirname "$0")/core.sh"
. "$(dirname "$0")/testnet-state.sh"

SIGNER_NAME="${SIGNER_NAME:=thorchain}"
SIGNER_PASSWD="${SIGNER_PASSWD:=password}"
NODES="${NODES:=1}"
SEED="${SEED:=thornode}" # the hostname of the master node
ETH_HOST="${ETH_HOST:=http://ethereum-localnet:8545}"

# this is required as it need to run thornode init , otherwise tendermint related commant doesn't work
if [ "$SEED" = "$(hostname)" ]; then
  if [ ! -f ~/.thornode/config/priv_validator_key.json ]; then
    init_chain
    # remove the original generate genesis file , as below will init chain again
    rm -rf ~/.thornode/config/genesis.json
  fi
fi

create_thor_user "$SIGNER_NAME" "$SIGNER_PASSWD" "$SIGNER_SEED_PHRASE"

VALIDATOR=$(thornode tendermint show-validator)
NODE_ADDRESS=$(echo "$SIGNER_PASSWD" | thornode keys show thorchain -a --keyring-backend file)
NODE_PUB_KEY=$(echo "$SIGNER_PASSWD" | thornode keys show thorchain -p --keyring-backend file)
VERSION=$(fetch_version)

mkdir -p /tmp/shared

if [ "$SEED" = "$(hostname)" ]; then
  thornode tendermint show-node-id >/tmp/shared/node.txt
fi

# write node account data to json file in shared directory
echo "$NODE_ADDRESS $VALIDATOR $NODE_PUB_KEY $VERSION $NODE_ADDRESS $NODE_PUB_KEY_ED25519" >"/tmp/shared/node_$NODE_ADDRESS.json"

# wait until THORNode have the correct number of nodes in our directory before continuing
while [ "$(find /tmp/shared -maxdepth 1 -type f -name 'node_*.json' | awk -F/ '{print $NF}' | wc -l | tr -d '[:space:]')" != "$NODES" ]; do
  sleep 1
done

if [ "$SEED" = "$(hostname)" ]; then
  echo "Setting THORNode as genesis"
  if [ ! -f ~/.thornode/config/genesis.json ]; then
    # get a list of addresses (thor bech32)
    ADDRS=""
    for f in /tmp/shared/node_*.json; do
      ADDRS="$ADDRS,$(awk <"$f" '{print $1}')"
    done
    init_chain "$(echo "$ADDRS" | sed -e 's/^,*//')"

    if [ -n "${VAULT_PUBKEY+x}" ]; then
      PUBKEYS=""
      for f in /tmp/shared/node_*.json; do
        PUBKEYS="$PUBKEYS,$(awk <"$f" '{print $3}')"
      done
      add_vault "$VAULT_PUBKEY" "$(echo "$PUBKEYS" | sed -e 's/^,*//')"
    fi

    NODE_IP_ADDRESS=${EXTERNAL_IP:=$(curl -s http://whatismyip.akamai.com)}

    # add node accounts to genesis file
    for f in /tmp/shared/node_*.json; do
      if [ -n "${VAULT_PUBKEY+x}" ]; then
        add_node_account "$(awk <"$f" '{print $1}')" "$(awk <"$f" '{print $2}')" "$(awk <"$f" '{print $3}')" "$(awk <"$f" '{print $4}')" "$(awk <"$f" '{print $5}')" "$(awk <"$f" '{print $6}')" "$NODE_IP_ADDRESS" "$VAULT_PUBKEY"
      else
        add_node_account "$(awk <"$f" '{print $1}')" "$(awk <"$f" '{print $2}')" "$(awk <"$f" '{print $3}')" "$(awk <"$f" '{print $4}')" "$(awk <"$f" '{print $5}')" "$(awk <"$f" '{print $6}')" "$NODE_IP_ADDRESS"
      fi
    done

    # add gases
    # add_gas_config "BNB.BNB" 37500 30000

    # disable default bank transfer, and opt to use our own custom one
    disable_bank_send

    # for mocknet, add heimdall balances
    echo "Using NET $NET"
    if [ "$NET" = "mocknet" ]; then
      echo "Setting up accounts"
      add_account tthor1z63f3mzwv3g75az80xwmhrawdqcjpaekk0kd54 rune 5000000000000
      add_account tthor1wz78qmrkplrdhy37tw0tnvn0tkm5pqd6zdp257 rune 25000000000100
      add_account tthor1xwusttz86hqfuk5z7amcgqsg7vp6g8zhsp5lu2 rune 5090000000000
      reserve 22000000000000000
      # deploy eth contract
      deploy_eth_contract $ETH_HOST
    else
      echo "ETH Contract Address: $CONTRACT"
      set_eth_contract "$CONTRACT"
    fi
    if [ "$NET" = "testnet" ]; then
      # mint 1m RUNE to reserve for testnet
      reserve 100000000000000

      # add existing account and balances
      add_exiting_accounts
    fi

    # enable telemetry through prometheus metrics endpoint
    enable_telemetry

    # enable internal traffic as well
    enable_internal_traffic

    # use external IP if available
    [ -n "$EXTERNAL_IP" ] && external_address "$EXTERNAL_IP" "$NET"

    echo "Genesis content"
    cat ~/.thornode/config/genesis.json
    thornode validate-genesis --trace
  fi
fi

# setup peer connection
if [ "$SEED" != "$(hostname)" ]; then
  if [ ! -f ~/.thornode/config/genesis.json ]; then
    echo "Setting THORNode as peer not genesis"

    init_chain "$NODE_ADDRESS"
    fetch_genesis $SEED
    NODE_ID=$(fetch_node_id $SEED)
    echo "NODE ID: $NODE_ID"
    peer_list "$NODE_ID" "$SEED"

    cat ~/.thornode/config/genesis.json
  fi
fi

printf "%s\n%s\n" "$SIGNER_NAME" "$SIGNER_PASSWD" | exec "$@"
