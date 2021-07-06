#!/bin/sh

# ./mock-leave.bash <mock binance IP address> <BNB Address> <THOR Address>
# ./mock-leave.bash 127.0.0.1 bnbXYXYX thorXXXX

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
  echo "Missing thor address argument"
  exit 1
fi

INBOUND_ADDRESS=$(curl -s "$1:1317/thorchain/inbound_addresses" | jq -r '.current[]|select(.chain=="BNB") .address')

curl -vvv -s -X POST -d "[{
  \"from\": \"$2\",
  \"to\": \"$INBOUND_ADDRESS\",
  \"coins\":[
      {\"denom\": \"RUNE-67C\", \"amount\": 1}
  ],
  \"memo\": \"LEAVE:$3\"
}]" "$1:26660/broadcast/easy"
