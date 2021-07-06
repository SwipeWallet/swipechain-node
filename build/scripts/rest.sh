#!/bin/sh

set -e

CHAIN_DAEMON="${CHAIN_DAEMON:=127.0.0.1:26657}"
echo $CHAIN_DAEMON

"$(dirname "$0")/wait-for-thorchain-daemon.sh" $CHAIN_DAEMON

exec "$@"
