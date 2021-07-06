package thorchain

import (
	"errors"

	. "gopkg.in/check.v1"

	"gitlab.com/thorchain/thornode/common"
	cosmos "gitlab.com/thorchain/thornode/common/cosmos"
	"gitlab.com/thorchain/thornode/constants"
	keeper "gitlab.com/thorchain/thornode/x/thorchain/keeper"
)

type SlashingSuite struct{}

var _ = Suite(&SlashingSuite{})

func (s *SlashingSuite) SetUpSuite(c *C) {
	SetupConfigForTest()
}

type TestSlashObservingKeeper struct {
	keeper.KVStoreDummy
	addrs                     []cosmos.AccAddress
	nas                       NodeAccounts
	failGetObservingAddress   bool
	failListActiveNodeAccount bool
	failSetNodeAccount        bool
	slashPts                  map[string]int64
}

func (k *TestSlashObservingKeeper) GetObservingAddresses(_ cosmos.Context) ([]cosmos.AccAddress, error) {
	if k.failGetObservingAddress {
		return nil, kaboom
	}
	return k.addrs, nil
}

func (k *TestSlashObservingKeeper) ClearObservingAddresses(_ cosmos.Context) {
	k.addrs = nil
}

func (k *TestSlashObservingKeeper) IncNodeAccountSlashPoints(_ cosmos.Context, addr cosmos.AccAddress, pts int64) error {
	if _, ok := k.slashPts[addr.String()]; !ok {
		k.slashPts[addr.String()] = 0
	}
	k.slashPts[addr.String()] += pts
	return nil
}

func (k *TestSlashObservingKeeper) ListActiveNodeAccounts(_ cosmos.Context) (NodeAccounts, error) {
	if k.failListActiveNodeAccount {
		return nil, kaboom
	}
	return k.nas, nil
}

func (k *TestSlashObservingKeeper) SetNodeAccount(_ cosmos.Context, na NodeAccount) error {
	if k.failSetNodeAccount {
		return kaboom
	}
	for i := range k.nas {
		if k.nas[i].NodeAddress.Equals(na.NodeAddress) {
			k.nas[i] = na
			return nil
		}
	}
	return errors.New("node account not found")
}

func (s *SlashingSuite) TestObservingSlashing(c *C) {
	var err error
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
	ver := constants.SWVersion
	constAccessor := constants.GetConstantValues(ver)

	slasher := NewSlasherV1(keeper)
	// should slash na2 only
	lackOfObservationPenalty := constAccessor.GetInt64Value(constants.LackOfObservationPenalty)
	err = slasher.LackObserving(ctx, constAccessor)
	c.Assert(err, IsNil)
	c.Assert(keeper.slashPts[nas[0].NodeAddress.String()], Equals, int64(0))
	c.Assert(keeper.slashPts[nas[1].NodeAddress.String()], Equals, lackOfObservationPenalty)

	// manually clear the observing address, as clear observing address had been moved to moduleManager begin block
	keeper.ClearObservingAddresses(ctx)
	// since THORNode have cleared all node addresses in slashForObservingAddresses,
	// running it a second time should result in slashing nobody.
	err = slasher.LackObserving(ctx, constAccessor)
	c.Assert(err, IsNil)
	c.Assert(keeper.slashPts[nas[0].NodeAddress.String()], Equals, int64(0))
	c.Assert(keeper.slashPts[nas[1].NodeAddress.String()], Equals, lackOfObservationPenalty)
}

func (s *SlashingSuite) TestLackObservingErrors(c *C) {
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
	ver := constants.SWVersion
	constAccessor := constants.GetConstantValues(ver)
	slasher := NewSlasherV1(keeper)
	keeper.failGetObservingAddress = true
	c.Assert(slasher.LackObserving(ctx, constAccessor), NotNil)
	keeper.failGetObservingAddress = false
	keeper.failListActiveNodeAccount = true
	c.Assert(slasher.LackObserving(ctx, constAccessor), NotNil)
	keeper.failListActiveNodeAccount = false
}

type TestSlashingLackKeeper struct {
	keeper.KVStoreDummy
	txOut                      *TxOut
	na                         NodeAccount
	vaults                     Vaults
	voter                      ObservedTxVoter
	failToGetAllPendingEvents  bool
	failGetTxOut               bool
	failGetVault               bool
	failGetNodeAccountByPubKey bool
	failSetNodeAccount         bool
	failGetAsgardByStatus      bool
	failGetObservedTxVoter     bool
	failSetTxOut               bool
	slashPts                   map[string]int64
}

func (k *TestSlashingLackKeeper) PoolExist(ctx cosmos.Context, asset common.Asset) bool {
	return true
}

func (k *TestSlashingLackKeeper) ListTxMarker(ctx cosmos.Context, hash string) (TxMarkers, error) {
	return TxMarkers{
		NewTxMarker(common.BlockHeight(ctx), "my memo"),
	}, nil
}

func (k *TestSlashingLackKeeper) GetObservedTxInVoter(_ cosmos.Context, _ common.TxID) (ObservedTxVoter, error) {
	if k.failGetObservedTxVoter {
		return ObservedTxVoter{}, kaboom
	}
	return k.voter, nil
}

func (k *TestSlashingLackKeeper) SetObservedTxInVoter(_ cosmos.Context, voter ObservedTxVoter) {
	k.voter = voter
}

func (k *TestSlashingLackKeeper) GetVault(_ cosmos.Context, pk common.PubKey) (Vault, error) {
	if k.failGetVault {
		return Vault{}, kaboom
	}
	return k.vaults[0], nil
}

func (k *TestSlashingLackKeeper) GetAsgardVaultsByStatus(_ cosmos.Context, _ VaultStatus) (Vaults, error) {
	if k.failGetAsgardByStatus {
		return nil, kaboom
	}
	return k.vaults, nil
}

func (k *TestSlashingLackKeeper) GetTxOut(_ cosmos.Context, _ int64) (*TxOut, error) {
	if k.failGetTxOut {
		return nil, kaboom
	}
	return k.txOut, nil
}

func (k *TestSlashingLackKeeper) SetTxOut(_ cosmos.Context, tx *TxOut) error {
	if k.failSetTxOut {
		return kaboom
	}
	k.txOut = tx
	return nil
}

func (k *TestSlashingLackKeeper) IncNodeAccountSlashPoints(_ cosmos.Context, addr cosmos.AccAddress, pts int64) error {
	if _, ok := k.slashPts[addr.String()]; !ok {
		k.slashPts[addr.String()] = 0
	}
	k.slashPts[addr.String()] += pts
	return nil
}

func (k *TestSlashingLackKeeper) GetNodeAccountByPubKey(_ cosmos.Context, _ common.PubKey) (NodeAccount, error) {
	if k.failGetNodeAccountByPubKey {
		return NodeAccount{}, kaboom
	}
	return k.na, nil
}

func (k *TestSlashingLackKeeper) SetNodeAccount(_ cosmos.Context, na NodeAccount) error {
	if k.failSetNodeAccount {
		return kaboom
	}
	k.na = na
	return nil
}

func (s *SlashingSuite) TestNodeSignSlashErrors(c *C) {
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
		ver := constants.SWVersion
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
		slasher := NewSlasherV1(keeper)
		item.condition(keeper)
		if item.shouldError {
			c.Assert(slasher.LackSigning(ctx, constAccessor, NewDummyMgr()), NotNil)
		} else {
			c.Assert(slasher.LackSigning(ctx, constAccessor, NewDummyMgr()), IsNil)
		}
	}
}

func (s *SlashingSuite) TestNotSigningSlash(c *C) {
	ctx, _ := setupKeeperForTest(c)
	ctx = ctx.WithBlockHeight(201) // set blockheight
	txOutStore := NewTxStoreDummy()
	ver := constants.SWVersion
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
	slasher := NewSlasherV1(keeper)
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

func (s *SlashingSuite) TestNewSlasher(c *C) {
	nas := NodeAccounts{
		GetRandomNodeAccount(NodeActive),
		GetRandomNodeAccount(NodeActive),
	}
	keeper := &TestSlashObservingKeeper{
		nas:      nas,
		addrs:    []cosmos.AccAddress{nas[0].NodeAddress},
		slashPts: make(map[string]int64, 0),
	}
	slasher := NewSlasherV1(keeper)
	c.Assert(slasher, NotNil)
}

type TestDoubleSlashKeeper struct {
	keeper.KVStoreDummy
	na          NodeAccount
	vaultData   VaultData
	slashPoints map[string]int64
	modules     map[string]int64
}

func (k *TestDoubleSlashKeeper) SendFromModuleToModule(_ cosmos.Context, from, to string, coin common.Coin) error {
	k.modules[from] -= int64(coin.Amount.Uint64())
	k.modules[to] += int64(coin.Amount.Uint64())
	return nil
}

func (k *TestDoubleSlashKeeper) ListActiveNodeAccounts(ctx cosmos.Context) (NodeAccounts, error) {
	return NodeAccounts{k.na}, nil
}

func (k *TestDoubleSlashKeeper) SetNodeAccount(ctx cosmos.Context, na NodeAccount) error {
	k.na = na
	return nil
}

func (k *TestDoubleSlashKeeper) GetVaultData(ctx cosmos.Context) (VaultData, error) {
	return k.vaultData, nil
}

func (k *TestDoubleSlashKeeper) SetVaultData(ctx cosmos.Context, data VaultData) error {
	k.vaultData = data
	return nil
}

func (k *TestDoubleSlashKeeper) IncNodeAccountSlashPoints(ctx cosmos.Context, addr cosmos.AccAddress, pts int64) error {
	k.slashPoints[addr.String()] += pts
	return nil
}

func (k *TestDoubleSlashKeeper) DecNodeAccountSlashPoints(ctx cosmos.Context, addr cosmos.AccAddress, pts int64) error {
	k.slashPoints[addr.String()] -= pts
	return nil
}

func (s *SlashingSuite) TestDoubleSign(c *C) {
	ctx, _ := setupKeeperForTest(c)
	constAccessor := constants.GetConstantValues(constants.SWVersion)

	na := GetRandomNodeAccount(NodeActive)
	na.Bond = cosmos.NewUint(100 * common.One)

	keeper := &TestDoubleSlashKeeper{
		na:        na,
		vaultData: NewVaultData(),
		modules:   make(map[string]int64, 0),
	}
	slasher := NewSlasherV1(keeper)

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

func (s *SlashingSuite) TestIncreaseDecreaseSlashPoints(c *C) {
	ctx, _ := setupKeeperForTest(c)

	na := GetRandomNodeAccount(NodeActive)
	na.Bond = cosmos.NewUint(100 * common.One)

	keeper := &TestDoubleSlashKeeper{
		na:          na,
		vaultData:   NewVaultData(),
		slashPoints: make(map[string]int64),
	}
	slasher := NewSlasherV1(keeper)
	addr := GetRandomBech32Addr()
	slasher.IncSlashPoints(ctx, 1, addr)
	slasher.DecSlashPoints(ctx, 1, addr)
	c.Assert(keeper.slashPoints[addr.String()], Equals, int64(0))
}
