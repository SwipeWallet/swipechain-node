#!/bin/sh

# ./mock-swap.bash <mock binance IP address> <bnb address> <amt> <asset> <swap to asset>
# ./mock-swap.bash 127.0.0.1 bnbZZZZZ 3000 RUNE-67C LOK-3C0

# set -e

if [ -z "$1" ]; then
  echo "Missing mock binance address (address:port)"
  exit 1
fi

if [ -z "$2" ]; then
  echo "Missing bnb address argument"
  exit 1
fi

if [ -z "$3" ]; then
  echo "Missing amount argument"
  exit 1
fi

if [ -z "$4" ]; then
  echo "Missing asset"
  exit 1
fi

if [ -z "$5" ]; then
  echo "Missing swap asset"
  exit 1
fi

INBOUND_ADDRESS=$(curl -s "$1:1317/thorchain/inbound_addresses" | jq -r '.current[]|select(.chain=="BNB") .address')

curl -s -X POST -d "[{
  \"from\": \"$2\",
  \"to\": \"$INBOUND_ADDRESS\",
  \"coins\":[
      {\"denom\": \"$4\", \"amount\": $3}
  ],
  \"memo\": \"SWAP:$5\"
}]" "$1:26660/broadcast/easy" | jq -r .hash
