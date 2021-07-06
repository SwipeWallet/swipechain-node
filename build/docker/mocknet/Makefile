mkfile_path := $(abspath $(lastword $(MAKEFILE_LIST)))
path := $(dir $(mkfile_path))

ifeq ($(OS),Windows_NT)
    uname_S := Windows
else
    uname_S := $(shell uname -s)
endif

run-mocknet-standalone:
	TAG=mocknet NET=mocknet docker-compose -p thornode -f ${path}../components/standalone.base.yml -f ${path}../components/frontend.yml -f ${path}../components/midgard.yml -f ${path}../components/midgard.${uname_S}.yml -f ${path}../components/mock-binance.yml -f ${path}../components/bitcoin-regtest.yml -f ${path}../components/ethereum-localnet.yml -f ${path}../mocknet/standalone.yml -f ${path}../components/standalone.${uname_S}.yml up --remove-orphans -d

reset-mocknet-standalone: stop clean run-mocknet-standalone

export-mocknet-standalone:
	TAG=mocknet NET=mocknet docker-compose -f ${path}../components/standalone.base.yml -f ${path}../mocknet/standalone.yml -f ${path}../components/standalone.${uname_S}.yml run thor-daemon sh -c "thord export > ~/.thord/genesis.json"

hard-fork-mocknet-standalone:
	TAG=mocknet NET=mocknet docker-compose -f ${path}../components/standalone.base.yml -f ${path}../mocknet/standalone.yml -f ${path}../components/standalone.${uname_S}.yml run thor-daemon sh -c "/docker/scripts/hard-fork.sh"

run-mocknet-genesis:
	TAG=mocknet NET=mocknet docker-compose -p thornode -f ${path}../components/genesis.base.yml -f ${path}../components/frontend.yml -f ${path}../components/midgard.yml -f ${path}../components/midgard.${uname_S}.yml -f ${path}../components/mock-binance.yml -f ${path}../mocknet/genesis.yml up --remove-orphans -d

run-mocknet-genesis-logs:
	TAG=mocknet NET=mocknet docker-compose -p thornode -f ${path}../components/genesis.base.yml -f ${path}../components/frontend.yml -f ${path}../components/midgard.yml -f ${path}../components/midgard.${uname_S}.yml -f ${path}../components/mock-binance.yml -f ${path}../mocknet/genesis.yml logs -f

reset-mocknet-genesis: stop clean run-mocknet-genesis

export-mocknet-genesis:
	TAG=mocknet NET=mocknet docker-compose -f ${path}../components/genesis.base.yml -f ${path}../mocknet/genesis.yml run thor-daemon sh -c "thord export > ~/.thord/genesis.json"

run-mocknet-genesis-churn:
	TAG=mocknet NET=mocknet docker-compose -p thornode -f ${path}../components/genesis.base.yml -f ${path}../components/fifthnode.yml -f ${path}../components/frontend.yml -f ${path}../components/midgard.yml -f ${path}../components/midgard.${uname_S}.yml -f ${path}../components/mock-binance.yml -f ${path}../mocknet/genesis.yml -f ${path}../mocknet/churn.yml up --remove-orphans -d

run-mocknet-genesis-churn-logs:
	TAG=mocknet NET=mocknet docker-compose -p thornode -f ${path}../components/genesis.base.yml -f ${path}../components/fifthnode.yml -f ${path}../components/frontend.yml -f ${path}../components/midgard.yml -f ${path}../components/midgard.${uname_S}.yml -f ${path}../components/mock-binance.yml -f ${path}../mocknet/genesis.yml -f ${path}../mocknet/churn.yml logs -f

reset-mocknet-genesis-churn: stop clean run-mocknet-genesis-churn

run-mocknet-validator:
	TAG=mocknet NET=mocknet docker-compose -p thornode -f ${path}../components/frontend.yml -f ${path}../components/midgard.yml -f ${path}../components/midgard.${uname_S}.yml -f ${path}../components/validator.yml -f ${path}../components/validator.${uname_S}.yml up --remove-orphans -d

reset-mocknet-validator: stop clean run-mocknet-validator

export-mocknet-validator:
	TAG=mocknet NET=mocknet docker-compose -f ${path}../components/validator.yml -f ${path}../components/validator.${uname_S}.yml run thor-daemon sh -c "thord export > ~/.thord/genesis.json"

hard-fork-mocknet-validator:
	TAG=mocknet NET=mocknet docker-compose -f ${path}../components/validator.yml -f ${path}../components/validator.${uname_S}.yml run thor-daemon sh -c "/docker/scripts/hard-fork.sh"
