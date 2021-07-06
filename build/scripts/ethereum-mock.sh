#!/bin/sh

if [ "$ETH_BLOCK_TIME" = "-1" ]; then
  ETH_BLOCK_TIME=5
fi

geth --dev --dev.period $ETH_BLOCK_TIME --verbosity 2 --networkid 15 --datadir "data" -mine --miner.threads 1 -rpc --rpcaddr 0.0.0.0 --rpcport 8545 --allow-insecure-unlock -nousb --rpcapi "eth,net,web3,miner,personal,txpool,debug" --rpccorsdomain "*" -nodiscover --rpcvhosts="*"
