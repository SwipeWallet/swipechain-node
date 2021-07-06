module gitlab.com/thorchain/thornode

go 1.14

require (
	github.com/99designs/keyring v1.1.3
	github.com/binance-chain/go-sdk v1.2.3
	github.com/binance-chain/ledger-cosmos-go v0.9.9 // indirect
	github.com/binance-chain/tss-lib v1.3.2
	github.com/blang/semver v3.5.1+incompatible
	github.com/btcsuite/btcd v0.20.1-beta.0.20200414114020-8b54b0b96418
	github.com/btcsuite/btcutil v1.0.2
	github.com/cenkalti/backoff v2.2.1+incompatible
	github.com/cosmos/cosmos-sdk v0.39.0
	github.com/cosmos/go-bip39 v0.0.0-20180819234021-555e2067c45d
	github.com/decred/dcrd/dcrec/edwards v1.0.0
	github.com/didip/tollbooth v4.0.2+incompatible
	github.com/ethereum/go-ethereum v1.9.12
	github.com/google/go-cmp v0.5.1 // indirect
	github.com/gorilla/mux v1.7.4
	github.com/hashicorp/go-multierror v1.1.0
	github.com/hashicorp/go-retryablehttp v0.6.4
	github.com/ipfs/go-log v1.0.4
	github.com/multiformats/go-multiaddr v0.3.1
	github.com/niemeyer/pretty v0.0.0-20200227124842-a10e7caefd8e // indirect
	github.com/patrickmn/go-cache v2.1.0+incompatible // indirect
	github.com/prometheus/client_golang v1.5.1
	github.com/prometheus/procfs v0.0.10 // indirect
	github.com/rjeczalik/notify v0.9.2 // indirect
	github.com/rs/zerolog v1.18.0
	github.com/spf13/afero v1.2.2 // indirect
	github.com/spf13/cast v1.3.1 // indirect
	github.com/spf13/cobra v1.0.0
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.6.3
	github.com/syndtr/goleveldb v1.0.1-0.20190923125748-758128399b1d
	github.com/tendermint/btcd v0.1.1
	github.com/tendermint/go-amino v0.15.1
	github.com/tendermint/tendermint v0.33.6
	github.com/tendermint/tm-db v0.5.1
	github.com/zondax/ledger-go v0.11.0 // indirect
	gitlab.com/thorchain/tss/go-tss v1.0.5-chaosnet
	gitlab.com/thorchain/txscript v0.0.0-20200413023754-8aaf3443d92b
	golang.org/x/lint v0.0.0-20200302205851-738671d3881b // indirect
	golang.org/x/tools v0.0.0-20200731060945-b5fad4ed8dd6 // indirect
	gopkg.in/check.v1 v1.0.0-20200227125254-8fa46927fb4f
	gopkg.in/ini.v1 v1.52.0 // indirect
)

replace (
	github.com/binance-chain/go-sdk => gitlab.com/thorchain/binance-sdk v1.2.2
	github.com/binance-chain/tss-lib => gitlab.com/thorchain/tss/tss-lib v0.0.0-20200723071108-d21a17ff2b2e
	github.com/tendermint/go-amino => github.com/binance-chain/bnc-go-amino v0.14.1-binance.1
)
