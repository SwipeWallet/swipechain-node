#!/bin/sh

set -e

HARDFORK_BLOCK_HEIGHT="${HARDFORK_BLOCK_HEIGHT:--1}"
DATE=$(date +%s)

# backup first
cp -r ~/.thornode/config ~/.thornode/config."$DATE".bak
cp -r ~/.thornode/data ~/.thornode/data."$DATE".bak

# export genesis file
thornode export --height "$HARDFORK_BLOCK_HEIGHT" >thorchain_genesis_export."$DATE".json

# reset the database
thornode unsafe-reset-all

# copied exported genesis file to the config directory
cp thorchain_genesis_export."$DATE".json ~/.thornode/config/genesis.json
