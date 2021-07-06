#!/bin/sh

# ./mock-unbond.bash <mock binance IP address> <BNB Address> <thor/node address> <thor API IP address>
# ./mock-unbond.bash 127.0.0.1 bnbXYXYX thor1kljxxccrheghavaw97u78le6yy3sdj7h696nl4

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
  echo "Missing node address argument (thor address)"
  exit 1
fi

if [ -z "$4" ]; then
  echo "Missing amount to unbond"
  exit 1
fi

INBOUND_ADDRESS=$(curl -s "$1:1317/thorchain/inbound_addresses" | jq -r '.current[]|select(.chain=="BNB") .address')

curl -v -s -X POST -d "[{
  \"from\": \"$2\",
  \"to\": \"$INBOUND_ADDRESS\",
  \"coins\":[
      {\"denom\": \"RUNE-67C\", \"amount\": 1}
  ],
  \"memo\": \"UNBOND:$3:$4\"
}]" "$1:26660/broadcast/easy"
