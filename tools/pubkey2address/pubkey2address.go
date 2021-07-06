package main

import (
	"flag"
	"fmt"

	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/common/cosmos"
)

func main() {
	raw := flag.String("p", "", "thor bech32 pubkey")
	flag.Parse()

	if len(*raw) == 0 {
		panic("no pubkey provided")
	}

	// Read in the configuration file for the sdk
	nw := common.GetCurrentChainNetwork()
	switch nw {
	case common.TestNet:
		fmt.Println("THORChain testnet:")
		config := cosmos.GetConfig()
		config.SetBech32PrefixForAccount("sswpe", "sswpepub")
		config.SetBech32PrefixForValidator("sswpev", "sswpevpub")
		config.SetBech32PrefixForConsensusNode("sswpec", "sswpecpub")
		config.Seal()
	case common.MainNet:
		fmt.Println("THORChain mainnet:")
		config := cosmos.GetConfig()
		config.SetBech32PrefixForAccount("swpe", "swpepub")
		config.SetBech32PrefixForValidator("swpev", "swpevpub")
		config.SetBech32PrefixForConsensusNode("swpec", "swpecpub")
		config.Seal()
	}

	pk, err := common.NewPubKey(*raw)
	if err != nil {
		panic(err)
	}

	chains := common.Chains{
		common.THORChain,
		common.BNBChain,
		common.BTCChain,
		common.ETHChain,
	}

	for _, chain := range chains {
		addr, err := pk.GetAddress(chain)
		if err != nil {
			panic(err)
		}
		fmt.Printf("%s Address: %s\n", chain.String(), addr)
	}
}
