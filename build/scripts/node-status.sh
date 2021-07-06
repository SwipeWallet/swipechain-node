#!/bin/sh

set -o pipefail

format_1e8() {
  printf "%.2f\n" "$(jq -n "$1"/100000000 2>/dev/null)" 2>/dev/null | sed ':a;s/\B[0-9]\{3\}\>/,&/;ta'
}

format_int() {
  printf "%.0f\n" "$1" 2>/dev/null | sed ':a;s/\B[0-9]\{3\}\>/,&/;ta'
}

API=http://thornode:1317
THORNODE_SERVICE_PORT_RPC="${THORNODE_SERVICE_PORT_RPC:=26657}"
BINANCE_DAEMON_SERVICE_PORT_RPC="${BINANCE_DAEMON_SERVICE_PORT_RPC:=26657}"
BITCOIN_DAEMON_SERVICE_PORT_RPC="${BITCOIN_DAEMON_SERVICE_PORT_RPC:=18443}"
LITECOIN_DAEMON_SERVICE_PORT_RPC="${LITECOIN_DAEMON_SERVICE_PORT_RPC:=18443}"
BITCOIN_CASH_DAEMON_SERVICE_PORT_RPC="${BITCOIN_CASH_DAEMON_SERVICE_PORT_RPC:=18443}"
ETHEREUM_DAEMON_SERVICE_PORT_RPC="${ETHEREUM_DAEMON_SERVICE_PORT_RPC:=8545}"

ADDRESS=$(echo "$SIGNER_PASSWD" | thornode keys show "$SIGNER_NAME" -a --keyring-backend file)
JSON=$(curl -sL --fail -m 10 "$API/thorchain/node/$ADDRESS")

IP=$(echo "$JSON" | jq -r ".ip_address")
VERSION=$(echo "$JSON" | jq -r ".version")
BOND=$(echo "$JSON" | jq -r ".bond")
REWARDS=$(echo "$JSON" | jq -r ".current_award")
SLASH=$(echo "$JSON" | jq -r ".slash_points")
STATUS=$(echo "$JSON" | jq -r ".status")
PREFLIGHT=$(echo "$JSON" | jq -r ".preflight_status")
[ "$VALIDATOR" = "false" ] && IP=$EXTERNAL_IP

if [ "$VALIDATOR" = "true" ]; then
  # calculate BNB chain sync progress
  [ "$NET" = "mainnet" ] && BNB_PEER=dataseed1.binance.org || BNB_PEER=data-seed-pre-0-s1.binance.org
  BNB_HEIGHT=$(curl -sL --fail -m 10 $BNB_PEER/status | jq -r ".result.sync_info.latest_block_height")
  BNB_SYNC_HEIGHT=$(curl -sL --fail -m 10 binance-daemon:$BINANCE_DAEMON_SERVICE_PORT_RPC/status | jq -r ".result.sync_info.index_height")
  BNB_PROGRESS=$(printf "%.3f%%" "$(jq -n "$BNB_SYNC_HEIGHT"/"$BNB_HEIGHT"*100 2>/dev/null)" 2>/dev/null) || BNB_PROGRESS=Error

  # calculate BTC chain sync progress
  BTC_RESULT=$(curl -sL --fail -m 10 --data-binary '{"jsonrpc": "1.0", "id": "node-status", "method": "getblockchaininfo", "params": []}' -H 'content-type: text/plain;' http://thorchain:password@bitcoin-daemon:$BITCOIN_DAEMON_SERVICE_PORT_RPC)
  BTC_HEIGHT=$(echo "$BTC_RESULT" | jq -r ".result.headers")
  BTC_SYNC_HEIGHT=$(echo "$BTC_RESULT" | jq -r ".result.blocks")
  BTC_PROGRESS=$(echo "$BTC_RESULT" | jq -r ".result.verificationprogress")
  BTC_PROGRESS=$(printf "%.3f%%" "$(jq -n "$BTC_PROGRESS"*100 2>/dev/null)" 2>/dev/null) || BTC_PROGRESS=Error

  # calculate LTC chain sync progress
  LTC_RESULT=$(curl -sL --fail -m 10 --data-binary '{"jsonrpc": "1.0", "id": "node-status", "method": "getblockchaininfo", "params": []}' -H 'content-type: text/plain;' http://thorchain:password@litecoin-daemon:$LITECOIN_DAEMON_SERVICE_PORT_RPC)
  LTC_HEIGHT=$(echo "$LTC_RESULT" | jq -r ".result.headers")
  LTC_SYNC_HEIGHT=$(echo "$LTC_RESULT" | jq -r ".result.blocks")
  LTC_PROGRESS=$(echo "$LTC_RESULT" | jq -r ".result.verificationprogress")
  LTC_PROGRESS=$(printf "%.3f%%" "$(jq -n "$LTC_PROGRESS"*100 2>/dev/null)" 2>/dev/null) || LTC_PROGRESS=Error

  # calculate ETH chain sync progress
  ETH_RESULT=$(curl -X POST -sL --fail -m 10 --data '{"jsonrpc":"2.0","method":"eth_syncing","params":[],"id":1}' -H 'content-type: application/json' http://ethereum-daemon:$ETHEREUM_DAEMON_SERVICE_PORT_RPC)
  if [ "$ETH_RESULT" = '{"jsonrpc":"2.0","id":1,"result":false}' ]; then
    ETH_RESULT=$(curl -X POST -sL --fail -m 10 --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' -H 'content-type: application/json' http://ethereum-daemon:$ETHEREUM_DAEMON_SERVICE_PORT_RPC)
    ETH_HEIGHT=$(printf "%.0f" "$(echo "$ETH_RESULT" | jq -r ".result")")
    ETH_SYNC_HEIGHT=$(printf "%.0f" "$(echo "$ETH_RESULT" | jq -r ".result")")
    ETH_PROGRESS=$(printf "%.3f%%" "$(jq -n "$ETH_SYNC_HEIGHT"/"$ETH_HEIGHT"*100 2>/dev/null)" 2>/dev/null) || ETH_PROGRESS=Error
  else
    ETH_HEIGHT=$(printf "%.0f" "$(echo "$ETH_RESULT" | jq -r ".result.highestBlock")")
    ETH_SYNC_HEIGHT=$(printf "%.0f" "$(echo "$ETH_RESULT" | jq -r ".result.currentBlock")")
    ETH_PROGRESS=$(printf "%.3f%%" "$(jq -n "$ETH_SYNC_HEIGHT"/"$ETH_HEIGHT"*100 2>/dev/null)" 2>/dev/null) || ETH_PROGRESS=Error
  fi

  # calculate BCH chain sync progress
  BCH_RESULT=$(curl -sL --fail -m 10 --data-binary '{"jsonrpc": "1.0", "id": "node-status", "method": "getblockchaininfo", "params": []}' -H 'content-type: text/plain;' http://thorchain:password@bitcoin-cash-daemon:$BITCOIN_CASH_DAEMON_SERVICE_PORT_RPC)
  BCH_HEIGHT=$(echo "$BCH_RESULT" | jq -r ".result.headers")
  BCH_SYNC_HEIGHT=$(echo "$BCH_RESULT" | jq -r ".result.blocks")
  BCH_PROGRESS=$(echo "$BCH_RESULT" | jq -r ".result.verificationprogress")
  BCH_PROGRESS=$(printf "%.3f%%" "$(jq -n "$BCH_PROGRESS"*100 2>/dev/null)" 2>/dev/null) || BCH_PROGRESS=Error
fi

# calculate THOR chain sync progress
THOR_SYNC_HEIGHT=$(curl -sL --fail -m 10 localhost:$THORNODE_SERVICE_PORT_RPC/status | jq -r ".result.sync_info.latest_block_height")
if [ "$PEER" != "" ]; then
  THOR_HEIGHT=$(curl -sL --fail -m 10 "$PEER:$THORNODE_SERVICE_PORT_RPC/status" | jq -r ".result.sync_info.latest_block_height")
elif [ "$SEEDS" != "" ]; then
  OLD_IFS=$IFS
  IFS=","
  for PEER in $SEEDS; do
    THOR_HEIGHT=$(curl -sL --fail -m 10 "$PEER:$THORNODE_SERVICE_PORT_RPC/status" | jq -r ".result.sync_info.latest_block_height") || continue
    break
  done
  IFS=$OLD_IFS
else
  THOR_HEIGHT=$THOR_SYNC_HEIGHT
fi
THOR_PROGRESS=$(printf "%.3f%%" "$(jq -n "$THOR_SYNC_HEIGHT"/"$THOR_HEIGHT"*100 2>/dev/null)" 2>/dev/null) || THOR_PROGRESS=Error

cat <<"EOF"
 ________ ______  ___  _  __        __
/_  __/ // / __ \/ _ \/ |/ /__  ___/ /__
 / / / _  / /_/ / , _/    / _ \/ _  / -_)
/_/ /_//_/\____/_/|_/_/|_/\___/\_,_/\__/
EOF
echo

if [ "$VALIDATOR" = "true" ]; then
  echo "ADDRESS     $ADDRESS"
  echo "IP          $IP"
  echo "VERSION     $VERSION"
  echo "STATUS      $STATUS"
  echo "BOND        $(format_1e8 "$BOND")"
  echo "REWARDS     $(format_1e8 "$REWARDS")"
  echo "SLASH       $(format_int "$SLASH")"
  echo "PREFLIGHT   $PREFLIGHT"
fi

echo
echo "API         http://$IP:1317/thorchain/doc/"
echo "RPC         http://$IP:$THORNODE_SERVICE_PORT_RPC"
echo "MIDGARD     http://$IP:8080/v2/doc"

echo
printf "%-11s %-10s %-10s\n" CHAIN SYNC BLOCKS
printf "%-11s %-10s %-10s\n" THOR "$THOR_PROGRESS" "$(format_int "$THOR_SYNC_HEIGHT")/$(format_int "$THOR_HEIGHT")"
[ "$VALIDATOR" = "true" ] && printf "%-11s %-10s %-10s\n" BNB "$BNB_PROGRESS" "$(format_int "$BNB_SYNC_HEIGHT")/$(format_int "$BNB_HEIGHT")"
[ "$VALIDATOR" = "true" ] && printf "%-11s %-10s %-10s\n" BTC "$BTC_PROGRESS" "$(format_int "$BTC_SYNC_HEIGHT")/$(format_int "$BTC_HEIGHT")"
[ "$VALIDATOR" = "true" ] && printf "%-11s %-10s %-10s\n" ETH "$ETH_PROGRESS" "$(format_int "$ETH_SYNC_HEIGHT")/$(format_int "$ETH_HEIGHT")"
[ "$VALIDATOR" = "true" ] && printf "%-11s %-10s %-10s\n" LTC "$LTC_PROGRESS" "$(format_int "$LTC_SYNC_HEIGHT")/$(format_int "$LTC_HEIGHT")"
[ "$VALIDATOR" = "true" ] && printf "%-11s %-10s %-10s\n" BCH "$BCH_PROGRESS" "$(format_int "$BCH_SYNC_HEIGHT")/$(format_int "$BCH_HEIGHT")"
exit 0
