#!/bin/sh

# ./mock-bond.bash <mock binance IP address> <BNB Address> <thor/node address> <thor API IP address>
# ./mock-bond.bash 127.0.0.1 bnbXYXYX thor1kljxxccrheghavaw97u78le6yy3sdj7h696nl4

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
  echo "Missing thor API address (IP/hostname)"
  exit 1
fi

curl -v -s -X POST -d "[{
  \"from\": \"tbnb1ht7v08hv2lhtmk8y7szl2hjexqryc3hcldlztl\",
  \"to\": \"$2\",
  \"coins\":[
      {\"denom\": \"BNB\", \"amount\": 200000000000000},
      {\"denom\": \"RUNE-67C\", \"amount\": 200000000000000}
  ],
  \"memo\": \"\"
}]" "$1/broadcast/easy"
