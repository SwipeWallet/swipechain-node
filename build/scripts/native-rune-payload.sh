#!/bin/sh

# ./native-rune-payload.sh <thor api address> <coins> <memo> <from address>

set -e

if [ -z "$1" ]; then
  echo "Missing thor api address"
  exit 1
fi

if [ -z "$2" ]; then
  echo "Missing coins"
  exit 1
fi

if [ -z "$3" ]; then
  echo "Missing memo"
  exit 1
fi

if [ -z "$4" ]; then
  echo "Missing from address"
  exit 1
fi

if [ -z "$5" ]; then
  echo "Missing local file name"
  exit 1
fi

curl -v -s -X POST -d "{
    \"coins\":$2,
    \"memo\": \"$3\",
    \"base_req\":
    {
        \"chain_id\": \"thorchain\",
        \"from\": \"$4\"
    }
}" "http://$1:1317/thorchain/deposit" -o "$5"
