package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	. "gopkg.in/check.v1"

	"gitlab.com/thorchain/thornode/common"
	cosmos "gitlab.com/thorchain/thornode/common/cosmos"
)

type MsgNetworkFeeSuite struct{}

var _ = Suite(&MsgNetworkFeeSuite{})

func (MsgNetworkFeeSuite) TestMsgNetworkFee(c *C) {
	msg := NewMsgNetworkFee(1024, common.BNBChain, 1, sdk.NewUint(37500), GetRandomBech32Addr())
	c.Assert(msg.Type(), Equals, "set_network_fee")
	EnsureMsgBasicCorrect(msg, c)

	testCases := []struct {
		blockHeight        int64
		name               string
		chain              common.Chain
		transactionSize    int64
		transactionFeeRate sdk.Uint
		signer             cosmos.AccAddress
		expectErr          bool
	}{
		{
			name:               "empty chain should return error",
			blockHeight:        1024,
			chain:              common.EmptyChain,
			transactionSize:    100,
			transactionFeeRate: sdk.NewUint(100),
			signer:             GetRandomBech32Addr(),
			expectErr:          true,
		},
		{
			name:               "invalid transaction size should return error",
			blockHeight:        1024,
			chain:              common.BNBChain,
			transactionSize:    -1,
			transactionFeeRate: sdk.NewUint(100),
			signer:             GetRandomBech32Addr(),
			expectErr:          true,
		},
		{
			name:               "invalid transaction fee rate should return error",
			blockHeight:        1024,
			chain:              common.BNBChain,
			transactionSize:    100,
			transactionFeeRate: sdk.ZeroUint(),
			signer:             GetRandomBech32Addr(),
			expectErr:          true,
		},
		{
			name:               "empty signer should return error",
			blockHeight:        1024,
			chain:              common.BNBChain,
			transactionSize:    100,
			transactionFeeRate: sdk.NewUint(100),
			signer:             cosmos.AccAddress(""),
			expectErr:          true,
		},
		{
			name:               "negative block height should return error",
			blockHeight:        -1024,
			chain:              common.BNBChain,
			transactionSize:    100,
			transactionFeeRate: sdk.NewUint(100),
			signer:             GetRandomBech32Addr(),
			expectErr:          true,
		},
		{
			name:               "happy path",
			blockHeight:        1024,
			chain:              common.BNBChain,
			transactionSize:    100,
			transactionFeeRate: sdk.NewUint(100),
			signer:             GetRandomBech32Addr(),
			expectErr:          false,
		},
	}
	for _, tc := range testCases {
		msg := NewMsgNetworkFee(tc.blockHeight, tc.chain, tc.transactionSize, tc.transactionFeeRate, tc.signer)

		err := msg.ValidateBasic()
		if tc.expectErr {
			c.Assert(err, NotNil, Commentf("name:%s", tc.name))
		} else {
			EnsureMsgBasicCorrect(msg, c)
		}

	}
}
