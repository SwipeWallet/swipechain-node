package thorchain

import (
	"github.com/blang/semver"
	. "gopkg.in/check.v1"

	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/common/cosmos"
	"gitlab.com/thorchain/thornode/constants"
)

type SlashingV17Suite struct{}

var _ = Suite(&SlashingV17Suite{})

func (s *SlashingV17Suite) SetUpSuite(c *C) {
	SetupConfigForTest()
}

func (s *SlashingV17Suite) TestObservingSlashing(c *C) {
	var err error
	ctx, k := setupKeeperForTest(c)
	naActiveAfterTx := GetRandomNodeAccount(NodeActive)
	naActiveAfterTx.ActiveBlockHeight = 1030
	nas := NodeAccounts{
		GetRandomNodeAccount(NodeActive),
		GetRandomNodeAccount(NodeActive),
		GetRandomNodeAccount(NodeStandby),
		naActiveAfterTx,
	}
	for _, item := range nas {
		c.Assert(k.SetNodeAccount(ctx, item), IsNil)
	}
	height := int64(1024)
	txOut := NewTxOut(height)
	txHash := GetRandomTxHash()
	observedTx := GetRandomObservedTx()
	txVoter := NewObservedTxVoter(txHash, []ObservedTx{
		observedTx,
	})
	txVoter.Height = 1024
	txVoter.Add(observedTx, nas[0].NodeAddress)
	k.SetObservedTxInVoter(ctx, txVoter)

	txOut.TxArray = append(txOut.TxArray, &TxOutItem{
		Chain:       common.BNBChain,
		InHash:      txHash,
		ToAddress:   GetRandomBNBAddress(),
		VaultPubKey: GetRandomPubKey(),
		Coin:        common.NewCoin(common.BNBAsset, cosmos.NewUint(1024)),
		Memo:        "whatever",
	})

	k.SetTxOut(ctx, txOut)

	ctx = ctx.WithBlockHeight(height + 300)
	ver := semver.MustParse("0.17.0")
	constAccessor := constants.GetConstantValues(ver)

	slasher := NewSlasherV17(k)
	// should slash na2 only
	lackOfObservationPenalty := constAccessor.GetInt64Value(constants.LackOfObservationPenalty)
	err = slasher.LackObserving(ctx, constAccessor)
	c.Assert(err, IsNil)
	slashPoint, err := k.GetNodeAccountSlashPoints(ctx, nas[0].NodeAddress)
	c.Assert(err, IsNil)
	c.Assert(slashPoint, Equals, int64(0))

	slashPoint, err = k.GetNodeAccountSlashPoints(ctx, nas[1].NodeAddress)
	c.Assert(err, IsNil)
	c.Assert(slashPoint, Equals, lackOfObservationPenalty)

	// standby node should not be slashed
	slashPoint, err = k.GetNodeAccountSlashPoints(ctx, nas[2].NodeAddress)
	c.Assert(err, IsNil)
	c.Assert(slashPoint, Equals, int64(0))

	// if node is active after the tx get observed , it should not be slashed
	slashPoint, err = k.GetNodeAccountSlashPoints(ctx, nas[3].NodeAddress)
	c.Assert(err, IsNil)
	c.Assert(slashPoint, Equals, int64(0))

	ctx = ctx.WithBlockHeight(height + 301)
	err = slasher.LackObserving(ctx, constAccessor)

	c.Assert(err, IsNil)
	slashPoint, err = k.GetNodeAccountSlashPoints(ctx, nas[0].NodeAddress)
	c.Assert(err, IsNil)
	c.Assert(slashPoint, Equals, int64(0))

	slashPoint, err = k.GetNodeAccountSlashPoints(ctx, nas[1].NodeAddress)
	c.Assert(err, IsNil)
	c.Assert(slashPoint, Equals, lackOfObservationPenalty)
}

func (s *SlashingV17Suite) TestLackObservingErrors(c *C) {
	ctx, _ := setupKeeperForTest(c)

	nas := NodeAccounts{
		GetRandomNodeAccount(NodeActive),
		GetRandomNodeAccount(NodeActive),
	}
	keeper := &TestSlashObservingKeeper{
		nas:      nas,
		addrs:    []cosmos.AccAddress{nas[0].NodeAddress},
		slashPts: make(map[string]int64, 0),
	}
	ver := semver.MustParse("0.17.0")
	constAccessor := constants.GetConstantValues(ver)
	slasher := NewSlasherV17(keeper)
	err := slasher.LackObserving(ctx, constAccessor)
	c.Assert(err, IsNil)
}

func (s *SlashingV17Suite) TestNodeSignSlashErrors(c *C) {
	testCases := []struct {
		name        string
		condition   func(keeper *TestSlashingLackKeeper)
		shouldError bool
	}{
		{
			name: "fail to get tx out should return an error",
			condition: func(keeper *TestSlashingLackKeeper) {
				keeper.failGetTxOut = true
			},
			shouldError: true,
		},
		{
			name: "fail to get vault should return an error",
			condition: func(keeper *TestSlashingLackKeeper) {
				keeper.failGetVault = true
			},
			shouldError: false,
		},
		{
			name: "fail to get node account by pub key should return an error",
			condition: func(keeper *TestSlashingLackKeeper) {
				keeper.failGetNodeAccountByPubKey = true
			},
			shouldError: false,
		},
		{
			name: "fail to get asgard vault by status should return an error",
			condition: func(keeper *TestSlashingLackKeeper) {
				keeper.failGetAsgardByStatus = true
			},
			shouldError: true,
		},
		{
			name: "fail to get observed tx voter should return an error",
			condition: func(keeper *TestSlashingLackKeeper) {
				keeper.failGetObservedTxVoter = true
			},
			shouldError: true,
		},
		{
			name: "fail to set tx out should return an error",
			condition: func(keeper *TestSlashingLackKeeper) {
				keeper.failSetTxOut = true
			},
			shouldError: true,
		},
	}
	for _, item := range testCases {
		c.Logf("name:%s", item.name)
		ctx, _ := setupKeeperForTest(c)
		ctx = ctx.WithBlockHeight(201) // set blockheight
		ver := semver.MustParse("0.17.0")
		constAccessor := constants.GetConstantValues(ver)
		na := GetRandomNodeAccount(NodeActive)
		inTx := common.NewTx(
			GetRandomTxHash(),
			GetRandomBNBAddress(),
			GetRandomBNBAddress(),
			common.Coins{
				common.NewCoin(common.BNBAsset, cosmos.NewUint(320000000)),
				common.NewCoin(common.RuneAsset(), cosmos.NewUint(420000000)),
			},
			nil,
			"SWAP:BNB.BNB",
		)

		txOutItem := &TxOutItem{
			Chain:       common.BNBChain,
			InHash:      inTx.ID,
			VaultPubKey: na.PubKeySet.Secp256k1,
			ToAddress:   GetRandomBNBAddress(),
			Coin: common.NewCoin(
				common.BNBAsset, cosmos.NewUint(3980500*common.One),
			),
		}
		txOut := NewTxOut(3)
		txOut.TxArray = append(txOut.TxArray, txOutItem)

		ygg := GetRandomVault()
		ygg.Type = YggdrasilVault
		keeper := &TestSlashingLackKeeper{
			txOut:  txOut,
			na:     na,
			vaults: Vaults{ygg},
			voter: ObservedTxVoter{
				Actions: []TxOutItem{*txOutItem},
			},
			slashPts: make(map[string]int64, 0),
		}
		signingTransactionPeriod := constAccessor.GetInt64Value(constants.SigningTransactionPeriod)
		ctx = ctx.WithBlockHeight(3 + signingTransactionPeriod)
		slasher := NewSlasherV17(keeper)
		item.condition(keeper)
		if item.shouldError {
			c.Assert(slasher.LackSigning(ctx, constAccessor, NewDummyMgr()), NotNil)
		} else {
			c.Assert(slasher.LackSigning(ctx, constAccessor, NewDummyMgr()), IsNil)
		}
	}
}

func (s *SlashingV17Suite) TestNotSigningSlash(c *C) {
	ctx, _ := setupKeeperForTest(c)
	ctx = ctx.WithBlockHeight(201) // set blockheight
	txOutStore := NewTxStoreDummy()
	ver := semver.MustParse("0.17.0")
	constAccessor := constants.GetConstantValues(ver)
	na := GetRandomNodeAccount(NodeActive)
	inTx := common.NewTx(
		GetRandomTxHash(),
		GetRandomBNBAddress(),
		GetRandomBNBAddress(),
		common.Coins{
			common.NewCoin(common.BNBAsset, cosmos.NewUint(320000000)),
			common.NewCoin(common.RuneAsset(), cosmos.NewUint(420000000)),
		},
		nil,
		"SWAP:BNB.BNB",
	)

	txOutItem := &TxOutItem{
		Chain:       common.BNBChain,
		InHash:      inTx.ID,
		VaultPubKey: na.PubKeySet.Secp256k1,
		ToAddress:   GetRandomBNBAddress(),
		Coin: common.NewCoin(
			common.BNBAsset, cosmos.NewUint(3980500*common.One),
		),
	}
	txOut := NewTxOut(3)
	txOut.TxArray = append(txOut.TxArray, txOutItem)

	ygg := GetRandomVault()
	ygg.Type = YggdrasilVault
	ygg.Coins = common.Coins{
		common.NewCoin(common.BNBAsset, cosmos.NewUint(5000000*common.One)),
	}
	keeper := &TestSlashingLackKeeper{
		txOut:  txOut,
		na:     na,
		vaults: Vaults{ygg},
		voter: ObservedTxVoter{
			Actions: []TxOutItem{*txOutItem},
		},
		slashPts: make(map[string]int64, 0),
	}
	signingTransactionPeriod := constAccessor.GetInt64Value(constants.SigningTransactionPeriod)
	ctx = ctx.WithBlockHeight(3 + signingTransactionPeriod)
	mgr := NewDummyMgr()
	mgr.txOutStore = txOutStore
	slasher := NewSlasherV17(keeper)
	c.Assert(slasher.LackSigning(ctx, constAccessor, mgr), IsNil)

	c.Check(keeper.slashPts[na.NodeAddress.String()], Equals, int64(600), Commentf("%+v\n", na))

	outItems, err := txOutStore.GetOutboundItems(ctx)
	c.Assert(err, IsNil)
	c.Assert(outItems, HasLen, 1)
	c.Assert(outItems[0].VaultPubKey.Equals(keeper.vaults[0].PubKey), Equals, true)
	c.Assert(outItems[0].Memo, Equals, "my memo")
	c.Assert(keeper.voter.Actions, HasLen, 1)
	// ensure we've updated our action item
	c.Assert(keeper.voter.Actions[0].VaultPubKey.Equals(outItems[0].VaultPubKey), Equals, true)
}

func (s *SlashingV17Suite) TestNewSlasher(c *C) {
	nas := NodeAccounts{
		GetRandomNodeAccount(NodeActive),
		GetRandomNodeAccount(NodeActive),
	}
	keeper := &TestSlashObservingKeeper{
		nas:      nas,
		addrs:    []cosmos.AccAddress{nas[0].NodeAddress},
		slashPts: make(map[string]int64, 0),
	}
	slasher := NewSlasherV17(keeper)
	c.Assert(slasher, NotNil)
}

func (s *SlashingV17Suite) TestDoubleSign(c *C) {
	ctx, _ := setupKeeperForTest(c)
	constAccessor := constants.GetConstantValues(constants.SWVersion)

	na := GetRandomNodeAccount(NodeActive)
	na.Bond = cosmos.NewUint(100 * common.One)

	keeper := &TestDoubleSlashKeeper{
		na:        na,
		vaultData: NewVaultData(),
		modules:   make(map[string]int64, 0),
	}
	slasher := NewSlasherV17(keeper)

	pk, err := cosmos.GetPubKeyFromBech32(cosmos.Bech32PubKeyTypeConsPub, na.ValidatorConsPubKey)
	c.Assert(err, IsNil)
	err = slasher.HandleDoubleSign(ctx, pk.Address(), 0, constAccessor)
	c.Assert(err, IsNil)

	c.Check(keeper.na.Bond.Equal(cosmos.NewUint(9995000000)), Equals, true, Commentf("%d", keeper.na.Bond.Uint64()))
	if common.RuneAsset().Chain.Equals(common.THORChain) {
		c.Check(keeper.modules[ReserveName], Equals, int64(5000000))
	} else {
		c.Check(keeper.vaultData.TotalReserve.Equal(cosmos.NewUint(5000000)), Equals, true)
	}
}

func (s *SlashingV17Suite) TestIncreaseDecreaseSlashPoints(c *C) {
	ctx, _ := setupKeeperForTest(c)

	na := GetRandomNodeAccount(NodeActive)
	na.Bond = cosmos.NewUint(100 * common.One)

	keeper := &TestDoubleSlashKeeper{
		na:          na,
		vaultData:   NewVaultData(),
		slashPoints: make(map[string]int64),
	}
	slasher := NewSlasherV17(keeper)
	addr := GetRandomBech32Addr()
	slasher.IncSlashPoints(ctx, 1, addr)
	slasher.DecSlashPoints(ctx, 1, addr)
	c.Assert(keeper.slashPoints[addr.String()], Equals, int64(0))
}
