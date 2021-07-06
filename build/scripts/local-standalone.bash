#!/bin/bash
# This script is designed to run a standalone swiped locally (non-containerized)

set -o pipefail

. "$(dirname "$0")/core.sh"

SIGNER_NAME="${SIGNER_NAME:=swipechain}"
SIGNER_PASSWD="${SIGNER_PASSWD:=password}"

create_thor_user "$SIGNER_NAME" "$SIGNER_PASSWD" "$SIGNER_SEED_PHRASE"

# if [ ! -f ~/.swiped/config/priv_validator_key.json ]; then
#   init_chain
#   # remove the original generate genesis file , as below will init chain again
#   rm -rf ~/.swiped/config/genesis.json
# fi

VALIDATOR=$(swiped tendermint show-validator)
echo $VALIDATOR
NODE_ADDRESS=$(echo "$SIGNER_PASSWD" | swipecli keys show swipechain -a --keyring-backend file)
NODE_PUB_KEY=$(echo "$SIGNER_PASSWD" | swipecli keys show swipechain -p --keyring-backend file)
VERSION=$(fetch_version)
NODE_IP_ADDRESS="127.0.0.1"

echo "Setting swiped as standalone"
if [ ! -f ~/.swiped/config/genesis.json ]; then
  init_chain

  add_node_account $NODE_ADDRESS $VALIDATOR $NODE_PUB_KEY $VERSION $NODE_ADDRESS $NODE_PUB_KEY $NODE_IP_ADDRESS

  # disable default bank transfer, and opt to use our own custom one
  disable_bank_send

  echo "Genesis content"
  cat ~/.swiped/config/genesis.json
  swiped validate-genesis --trace
fi

printf "%s\n%s\n" "$SIGNER_NAME" "$SIGNER_PASSWD" | swiped start --rpc.laddr "tcp://0.0.0.0:26657"
