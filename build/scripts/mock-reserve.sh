#!/bin/sh

# ./mock-reserve.bash <mock binance IP address> <from_address> <num of rune>
# ./mock-reserve.bash 127.0.0.1 bnbZXXXX 22000000000000000

set -e

if [ -z "$1" ]; then
  echo "Missing mock binance address (address:port)"
  exit 1
fi

if [ -z "$2" ]; then
  echo "Missing bnb address argument"
  exit 1
fi

if [ -z "$3" ]; then
  echo "Missing rune amount"
  exit 1
fi

INBOUND_ADDRESS=$(curl -s "$1:1317/thorchain/INBOUND_addresses" | jq -r ".current[0].address")
echo "$POOL_ADDRESS"

curl -s -X POST -d "[{
  \"from\": \"$2\",
  \"to\": \"$INBOUND_ADDRESS\",
  \"coins\":[
      {\"denom\": \"RUNE-A1F\", \"amount\": $3}
  ],
  \"memo\": \"RESERVE\"
}]" "$1:26660/broadcast/easy"
