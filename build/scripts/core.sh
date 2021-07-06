#!/bin/sh

set -o pipefail

PORT_P2P=26656
PORT_RPC=26657
[ "$NET" = "mainnet" ] && PORT_P2P=27146 && PORT_RPC=27147

# adds an account node into the genesis file
add_node_account() {
  NODE_ADDRESS=$1
  VALIDATOR=$2
  NODE_PUB_KEY=$3
  VERSION=$4
  BOND_ADDRESS=$5
  NODE_PUB_KEY_ED25519=$6
  IP_ADDRESS=$7
  MEMBERSHIP=$8
  jq --arg IP_ADDRESS "$IP_ADDRESS" --arg VERSION "$VERSION" --arg BOND_ADDRESS "$BOND_ADDRESS" --arg VALIDATOR "$VALIDATOR" --arg NODE_ADDRESS "$NODE_ADDRESS" --arg NODE_PUB_KEY "$NODE_PUB_KEY" --arg NODE_PUB_KEY_ED25519 "$NODE_PUB_KEY_ED25519" '.app_state.thorchain.node_accounts += [{"node_address": $NODE_ADDRESS, "version": $VERSION, "ip_address": $IP_ADDRESS, "status": "Active", "active_block_height": "0", "bond_address":$BOND_ADDRESS, "signer_membership": [], "validator_cons_pub_key":$VALIDATOR, "pub_key_set":{"secp256k1":$NODE_PUB_KEY,"ed25519":$NODE_PUB_KEY_ED25519}}]' <~/.swiped/config/genesis.json >/tmp/genesis.json
  mv /tmp/genesis.json ~/.swiped/config/genesis.json
  if [ -n "$MEMBERSHIP" ]; then
    jq --arg MEMBERSHIP "$MEMBERSHIP" '.app_state.thorchain.node_accounts[-1].signer_membership += [$MEMBERSHIP]' ~/.swiped/config/genesis.json >/tmp/genesis.json
    mv /tmp/genesis.json ~/.swiped/config/genesis.json
  fi
}

add_last_event_id() {
  echo "Adding last event id $1"
  jq --arg ID "$1" '.app_state.thorchain.last_event_id = $ID' ~/.thornode/config/genesis.json >/tmp/genesis.json
  mv /tmp/genesis.json ~/.thornode/config/genesis.json
}

add_gas_config() {
  asset=$1
  shift

  # add asset to gas
  jq --argjson path "[\"app_state\", \"thorchain\", \"gas\", \"$asset\"]" 'getpath($path) = []' ~/.thornode/config/genesis.json >/tmp/genesis.json
  mv /tmp/genesis.json ~/.thornode/config/genesis.json

  for unit in "$@"; do
    jq --argjson path "[\"app_state\", \"thorchain\", \"gas\", \"$asset\"]" --arg unit "$unit" 'getpath($path) += [$unit]' ~/.thornode/config/genesis.json >/tmp/genesis.json
    mv /tmp/genesis.json ~/.thornode/config/genesis.json
  done
}

reserve() {
  jq --arg RESERVE "$1" '.app_state.thorchain.reserve = $RESERVE' <~/.thornode/config/genesis.json >/tmp/genesis.json
  mv /tmp/genesis.json ~/.thornode/config/genesis.json
}

disable_bank_send() {
  jq '.app_state.bank.params.default_send_enabled = false' <~/.swiped/config/genesis.json >/tmp/genesis.json
  mv /tmp/genesis.json ~/.swiped/config/genesis.json

  jq '.app_state.transfer.params.send_enabled = false' <~/.swiped/config/genesis.json >/tmp/genesis.json
  mv /tmp/genesis.json ~/.swiped/config/genesis.json
}

add_account() {
  jq --arg ADDRESS "$1" --arg ASSET "$2" --arg AMOUNT "$3" '.app_state.auth.accounts += [{
        "@type": "/cosmos.auth.v1beta1.BaseAccount",
        "address": $ADDRESS,
        "pub_key": null,
        "account_number": "0",
        "sequence": "0"
    }]' <~/.thornode/config/genesis.json >/tmp/genesis.json
  # "coins": [ { "denom": $ASSET, "amount": $AMOUNT } ],
  mv /tmp/genesis.json ~/.thornode/config/genesis.json

  jq --arg ADDRESS "$1" --arg ASSET "$2" --arg AMOUNT "$3" '.app_state.bank.balances += [{
        "address": $ADDRESS,
        "coins": [ { "denom": $ASSET, "amount": $AMOUNT } ],
    }]' <~/.thornode/config/genesis.json >/tmp/genesis.json
  mv /tmp/genesis.json ~/.thornode/config/genesis.json
}

add_vault() {
  POOL_PUBKEY=$1
  shift

  jq --arg POOL_PUBKEY "$POOL_PUBKEY" '.app_state.thorchain.vaults += [{"block_height": "0", "pub_key": $POOL_PUBKEY, "coins":[], "type": "asgard", "status":"active", "status_since": "0", "membership":[]}]' <~/.thornode/config/genesis.json >/tmp/genesis.json
  mv /tmp/genesis.json ~/.thornode/config/genesis.json

  export IFS=","
  for pubkey in "$@"; do # iterate over our list of comma separated pubkeys
    jq --arg PUBKEY "$pubkey" '.app_state.thorchain.vaults[0].membership += [$PUBKEY]' ~/.thornode/config/genesis.json >/tmp/genesis.json
    mv /tmp/genesis.json ~/.thornode/config/genesis.json
  done
}

# inits a thorchain with a comman separate list of usernames
init_chain() {
  export IFS=","

  echo "Init chain"
  swiped init local --chain-id swipechain
  echo "$SIGNER_PASSWD" | swipecli keys list --keyring-backend file

  for user in "$@"; do # iterate over our list of comma separated users "alice,jack"
    swiped add-genesis-account "$user" 100000000sxp
  done

  # thornode config chain-id thorchain
  # thornode config output json
  # thornode config indent true
  # thornode config trust-node true
}

peer_list() {
  PEERUSER="$1@$2:$PORT_P2P"
  PEERSISTENT_PEER_TARGET='persistent_peers = ""'
  sed -i -e "s/$PEERSISTENT_PEER_TARGET/persistent_peers = \"$PEERUSER\"/g" ~/.thornode/config/config.toml
}

seeds_list() {
  SEEDS=$1
  OLD_IFS=$IFS
  IFS=","
  SEED_LIST=""
  for SEED in $SEEDS; do
    NODE_ID=$(curl -sL --fail -m 10 "$SEED:$PORT_RPC/status" | jq -r .result.node_info.id) || continue
    SEED="$NODE_ID@$SEED:$PORT_P2P"
    if [ -z "$SEED_LIST" ]; then
      SEED_LIST=$SEED
    else
      SEED_LIST="$SEED_LIST,$SEED"
    fi
  done
  IFS=$OLD_IFS
  sed -i -e "s/seeds =.*/seeds = \"$SEED_LIST\"/g" ~/.thornode/config/config.toml
}

enable_internal_traffic() {
  ADDR='addr_book_strict = true'
  ADDR_STRICT_FALSE='addr_book_strict = false'
  sed -i -e "s/$ADDR/$ADDR_STRICT_FALSE/g" ~/.thornode/config/config.toml
}

external_address() {
  IP=$1
  NET=$2
  ADDR="$IP:$PORT_P2P"
  sed -i -e "s/external_address =.*/external_address = \"$ADDR\"/g" ~/.thornode/config/config.toml
}

enable_telemetry() {
  sed -i -e "s/prometheus = false/prometheus = true/g" ~/.thornode/config/config.toml
}

gen_bnb_address() {
  if [ ! -f ~/.bond/private_key.txt ]; then
    echo "Generating BNB address"
    mkdir -p ~/.bond
    # because the generate command can get API rate limited, THORNode may need to retry
    n=0
    until [ $n -ge 60 ]; do
      generate >/tmp/bnb && break
      n=$((n + 1))
      sleep 1
    done
    ADDRESS=$(grep </tmp/bnb MASTER= | awk -F= '{print $NF}')
    echo "$ADDRESS" >~/.bond/address.txt
    BINANCE_PRIVATE_KEY=$(grep </tmp/bnb MASTER_KEY= | awk -F= '{print $NF}')
    echo "$BINANCE_PRIVATE_KEY" >/root/.bond/private_key.txt
    PUBKEY=$(grep </tmp/bnb MASTER_PUBKEY= | awk -F= '{print $NF}')
    echo "$PUBKEY" >/root/.bond/pubkey.txt
    MNEMONIC=$(grep </tmp/bnb MASTER_MNEMONIC= | awk -F= '{print $NF}')
    echo "$MNEMONIC" >/root/.bond/mnemonic.txt
  fi
}

deploy_eth_contract() {
  echo "Deploying eth contracts"
  until curl -s "$1" &>/dev/null; do
    echo "Waiting for ETH node to be available ($1)"
    sleep 3
  done
  python3 scripts/eth/eth-tool.py --ethereum "$1" deploy --from_address 0x3fd2d4ce97b082d4bce3f9fee2a3d60668d2f473 >/tmp/contract.log 2>&1
  cat /tmp/contract.log
  CONTRACT=$(grep </tmp/contract.log "Vault Contract Address" | awk '{print $NF}')
  echo "Contract Address: $CONTRACT"

  set_eth_contract "$CONTRACT"
}

set_eth_contract() {
  jq --arg CONTRACT "$1" '.app_state.thorchain.chain_contracts = [{"chain": "ETH", "router": $CONTRACT}]' ~/.thornode/config/genesis.json >/tmp/genesis.json
  mv /tmp/genesis.json ~/.thornode/config/genesis.json
}

fetch_genesis() {
  echo "Fetching genesis"
  until curl -s "$1:$PORT_RPC" &>/dev/null; do
    sleep 3
  done
  curl -s "$1:$PORT_RPC/genesis" | jq .result.genesis >~/.thornode/config/genesis.json
  thornode validate-genesis --trace
  cat ~/.thornode/config/genesis.json
}

fetch_genesis_from_seeds() {
  SEEDS=$1
  OLD_IFS=$IFS
  IFS=","
  SEED_LIST=""
  for SEED in $SEEDS; do
    echo "Fetching genesis from seed $SEED"
    curl -sL --fail -m 10 "$SEED:$PORT_RPC/genesis" | jq .result.genesis >~/.thornode/config/genesis.json || continue
    thornode validate-genesis
    cat ~/.thornode/config/genesis.json
    break
  done
  IFS=$OLD_IFS
}

fetch_node_id() {
  until curl -s "$1:$PORT_RPC" &>/dev/null; do
    sleep 3
  done
  curl -s "$1:$PORT_RPC/status" | jq -r .result.node_info.id
}

set_node_keys() {
  SIGNER_NAME="$1"
  SIGNER_PASSWD="$2"
  PEER="$3"
  NODE_PUB_KEY="$(echo "$SIGNER_PASSWD" | thornode keys show thorchain --pubkey --keyring-backend file)"
  NODE_PUB_KEY_ED25519="$(printf "%s\n" "$SIGNER_PASSWD" | thornode ed25519)"
  VALIDATOR="$(thornode tendermint show-validator)"
  echo "Setting THORNode keys"
  printf "%s\n%s\n" "$SIGNER_PASSWD" "$SIGNER_PASSWD" | thornode tx thorchain set-node-keys "$NODE_PUB_KEY" "$NODE_PUB_KEY_ED25519" "$VALIDATOR" --node "tcp://$PEER:$PORT_RPC" --from "$SIGNER_NAME" --yes
}

set_ip_address() {
  SIGNER_NAME="$1"
  SIGNER_PASSWD="$2"
  PEER="$3"
  NODE_IP_ADDRESS="${4:-$(curl -s http://whatismyip.akamai.com)}"
  echo "Setting THORNode IP address $NODE_IP_ADDRESS"
  printf "%s\n%s\n" "$SIGNER_PASSWD" "$SIGNER_PASSWD" | thornode tx thorchain set-ip-address "$NODE_IP_ADDRESS" --node "tcp://$PEER:$PORT_RPC" --from "$SIGNER_NAME" --yes
}

fetch_version() {
  swipecli query thorchain version --output json | jq -r .version
}

create_thor_user() {
  SIGNER_NAME="$1"
  SIGNER_PASSWD="$2"
  SIGNER_SEED_PHRASE="$3"

  echo "Checking if SWIPENode '$SIGNER_NAME' account exists"
  echo "$SIGNER_PASSWD" | swipecli keys show "$SIGNER_NAME" --keyring-backend file &>/dev/null
  # shellcheck disable=SC2181
  if [ $? -ne 0 ]; then
    echo "Creating SWIPENode '$SIGNER_NAME' account"
    if [ -n "$SIGNER_SEED_PHRASE" ]; then
      printf "%s\n%s\n%s\n" "$SIGNER_SEED_PHRASE" "$SIGNER_PASSWD" "$SIGNER_PASSWD" | swipecli keys --keyring-backend file add "$SIGNER_NAME" --recover
    else
      sig_pw=$(printf "%s\n%s\n" "$SIGNER_PASSWD" "$SIGNER_PASSWD")
      RESULT=$(echo "$sig_pw" | swipecli keys --keyring-backend file add "$SIGNER_NAME" --output json 2>&1)
      SIGNER_SEED_PHRASE=$(echo "$RESULT" | jq -r '.mnemonic')
    fi
  fi
  NODE_PUB_KEY_ED25519=$(printf "%s\n%s\n" "$SIGNER_PASSWD" "$SIGNER_SEED_PHRASE" | swiped ed25519)
}
