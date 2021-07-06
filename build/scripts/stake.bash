#!/bin/bash

# ./mock-bond.bash <mock binance IP address> <thor/node address>
# ./mock-bond.bash 127.0.0.1 thor1kljxxccrheghavaw97u78le6yy3sdj7h696nl4

# set -e

if [ -z "$1" ]; then
  echo "Missing mock binance address (address:port)"
  exit 1
fi

if [ -z "$2" ]; then
  echo "Missing node address argument (thor address)"
  exit 1
fi

INBOUND_ADDRESS=$(curl -s "$1:1317/thorchain/inbound_addresses" | jq -r '.current[]|select(.chain=="BNB") .address')

# NOTE: the from address doesn't matter at all (mock binance doesn't care)

curl -vvv -s -X POST -d "{
  \"from\": \"tbnb1rlmrd83gv7rk2thusqm7dx38z8jgur80t8kq28\",
  \"to\": \"$INBOUND_ADDRESS\",
  \"coins\":[
      {\"denom\": \"BNB.RUNE-67C\", \"amount\": 100000000000},
      {\"denom\": \"BNB.BNB\", \"amount\": 7000000000}
  ],
  \"memo\": \"STAKE:BNB.BNB\"
}" "$1:26660/broadcast/easy"
