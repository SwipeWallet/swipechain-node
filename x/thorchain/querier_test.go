package thorchain

import (
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	abci "github.com/tendermint/tendermint/abci/types"
	. "gopkg.in/check.v1"

	ckeys "github.com/cosmos/cosmos-sdk/crypto/keys"
	"github.com/cosmos/cosmos-sdk/crypto/keys/hd"

	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/common/cosmos"
	"gitlab.com/thorchain/thornode/x/thorchain/keeper"
	"gitlab.com/thorchain/thornode/x/thorchain/query"
	"gitlab.com/thorchain/thornode/x/thorchain/types"
)

type QuerierSuite struct {
	kb      KeybaseStore
	k       keeper.Keeper
	querier cosmos.Querier
	ctx     cosmos.Context
}

var _ = Suite(&QuerierSuite{})

type TestQuerierKeeper struct {
	keeper.KVStoreDummy
	txOut *TxOut
}

func (k *TestQuerierKeeper) GetTxOut(_ cosmos.Context, _ int64) (*TxOut, error) {
	return k.txOut, nil
}

func (s *QuerierSuite) SetUpTest(c *C) {
	kb := ckeys.NewInMemory()
	username := "test_user"
	password := "password"

	params := *hd.NewFundraiserParams(0, 118, 0)
	hdPath := params.String()
	_, err := kb.CreateAccount(username, "industry segment educate height inject hover bargain offer employ select speak outer video tornado story slow chief object junk vapor venue large shove behave", password, password, hdPath, ckeys.Secp256k1)
	c.Assert(err, IsNil)
	s.kb = KeybaseStore{
		SignerName:   username,
		SignerPasswd: password,
		Keybase:      kb,
	}
	ctx, k := setupKeeperForTest(c)
	s.k = k
	s.ctx = ctx
	s.querier = NewQuerier(k, s.kb)
}

func (s *QuerierSuite) TestQueryKeysign(c *C) {
	ctx, _ := setupKeeperForTest(c)
	ctx = ctx.WithBlockHeight(12)

	pk := GetRandomPubKey()
	toAddr := GetRandomBNBAddress()
	txOut := NewTxOut(1)
	txOutItem := &TxOutItem{
		Chain:       common.BNBChain,
		VaultPubKey: pk,
		ToAddress:   toAddr,
		InHash:      GetRandomTxHash(),
		Coin:        common.NewCoin(common.BNBAsset, cosmos.NewUint(100*common.One)),
	}
	txOut.TxArray = append(txOut.TxArray, txOutItem)
	keeper := &TestQuerierKeeper{
		txOut: txOut,
	}

	querier := NewQuerier(keeper, s.kb)

	path := []string{
		"keysign",
		"5",
		pk.String(),
	}
	res, err := querier(ctx, path, abci.RequestQuery{})
	c.Assert(err, IsNil)
	c.Assert(res, NotNil)
}

func (s *QuerierSuite) TestQueryPool(c *C) {
	ctx, keeper := setupKeeperForTest(c)

	querier := NewQuerier(keeper, s.kb)
	path := []string{"pools"}

	pubKey := GetRandomPubKey()
	asgard := NewVault(common.BlockHeight(ctx), ActiveVault, AsgardVault, pubKey, common.Chains{common.BNBChain})
	c.Assert(keeper.SetVault(ctx, asgard), IsNil)

	poolBNB := NewPool()
	poolBNB.Asset = common.BNBAsset
	poolBNB.PoolUnits = cosmos.NewUint(100)

	poolBTC := NewPool()
	poolBTC.Asset = common.BTCAsset
	poolBTC.PoolUnits = cosmos.NewUint(0)

	err := keeper.SetPool(ctx, poolBNB)
	c.Assert(err, IsNil)

	err = keeper.SetPool(ctx, poolBTC)
	c.Assert(err, IsNil)

	res, err := querier(ctx, path, abci.RequestQuery{})
	c.Assert(err, IsNil)

	var out Pools
	err = keeper.Cdc().UnmarshalJSON(res, &out)
	c.Assert(err, IsNil)
	c.Assert(len(out), Equals, 1)

	poolBTC.PoolUnits = cosmos.NewUint(100)
	err = keeper.SetPool(ctx, poolBTC)
	c.Assert(err, IsNil)

	res, err = querier(ctx, path, abci.RequestQuery{})
	c.Assert(err, IsNil)

	err = keeper.Cdc().UnmarshalJSON(res, &out)
	c.Assert(err, IsNil)
	c.Assert(len(out), Equals, 2)

	result, err := s.querier(s.ctx, []string{query.QueryPool.Key, "BNB.BNB"}, abci.RequestQuery{})
	c.Assert(result, HasLen, 0)
	c.Assert(err, NotNil)
}

func (s *QuerierSuite) TestQueryNodeAccounts(c *C) {
	ctx, keeper := setupKeeperForTest(c)

	querier := NewQuerier(keeper, s.kb)
	path := []string{"nodeaccounts"}

	signer := GetRandomBech32Addr()
	bondAddr := GetRandomBNBAddress()
	emptyPubKeySet := common.PubKeySet{}
	bond := cosmos.NewUint(common.One * 100)
	nodeAccount := NewNodeAccount(signer, NodeActive, emptyPubKeySet, "", bond, bondAddr, common.BlockHeight(ctx))
	c.Assert(keeper.SetNodeAccount(ctx, nodeAccount), IsNil)

	res, err := querier(ctx, path, abci.RequestQuery{})
	c.Assert(err, IsNil)

	var out types.NodeAccounts
	err1 := keeper.Cdc().UnmarshalJSON(res, &out)
	c.Assert(err1, IsNil)
	c.Assert(len(out), Equals, 1)

	signer = GetRandomBech32Addr()
	bondAddr = GetRandomBNBAddress()
	emptyPubKeySet = common.PubKeySet{}
	bond = cosmos.NewUint(common.One * 200)
	nodeAccount2 := NewNodeAccount(signer, NodeActive, emptyPubKeySet, "", bond, bondAddr, common.BlockHeight(ctx))
	c.Assert(keeper.SetNodeAccount(ctx, nodeAccount2), IsNil)

	res, err = querier(ctx, path, abci.RequestQuery{})
	c.Assert(err, IsNil)

	err1 = keeper.Cdc().UnmarshalJSON(res, &out)
	c.Assert(err1, IsNil)
	c.Assert(len(out), Equals, 2)

	nodeAccount2.Bond = cosmos.NewUint(0)
	c.Assert(keeper.SetNodeAccount(ctx, nodeAccount2), IsNil)

	res, err = querier(ctx, path, abci.RequestQuery{})
	c.Assert(err, IsNil)

	err1 = keeper.Cdc().UnmarshalJSON(res, &out)
	c.Assert(err1, IsNil)
	c.Assert(len(out), Equals, 1)
}

func (s *QuerierSuite) TestQuerierRagnarokInProgress(c *C) {
	req := abci.RequestQuery{
		Data:   nil,
		Path:   query.QueryRagnarok.Key,
		Height: s.ctx.BlockHeight(),
		Prove:  false,
	}
	// test ragnarok
	result, err := s.querier(s.ctx, []string{query.QueryRagnarok.Key}, req)
	c.Assert(result, NotNil)
	c.Assert(err, IsNil)
	var ragnarok bool
	c.Assert(s.k.Cdc().UnmarshalJSON(result, &ragnarok), IsNil)
	c.Assert(ragnarok, Equals, false)
}

func (s *QuerierSuite) TestQueryStakers(c *C) {
	req := abci.RequestQuery{
		Data:   nil,
		Path:   query.QueryStakers.Key,
		Height: s.ctx.BlockHeight(),
		Prove:  false,
	}
	// test stakers
	result, err := s.querier(s.ctx, []string{query.QueryStakers.Key, "BNB.BNB"}, req)
	c.Assert(result, NotNil)
	c.Assert(err, IsNil)
	s.k.SetStaker(s.ctx, Staker{
		Asset:             common.BNBAsset,
		RuneAddress:       GetRandomBNBAddress(),
		AssetAddress:      GetRandomBNBAddress(),
		LastStakeHeight:   1024,
		LastUnStakeHeight: 0,
		Units:             cosmos.NewUint(10),
	})
	result, err = s.querier(s.ctx, []string{query.QueryStakers.Key, "BNB.BNB"}, req)
	c.Assert(err, IsNil)
	var stakers []Staker
	c.Assert(s.k.Cdc().UnmarshalJSON(result, &stakers), IsNil)
	c.Assert(stakers, HasLen, 1)
}

func (s *QuerierSuite) TestQueryTxInVoter(c *C) {
	req := abci.RequestQuery{
		Data:   nil,
		Path:   query.QueryTxInVoter.Key,
		Height: s.ctx.BlockHeight(),
		Prove:  false,
	}
	tx := GetRandomTx()
	// test getTxInVoter
	result, err := s.querier(s.ctx, []string{query.QueryTxInVoter.Key, tx.ID.String()}, req)
	c.Assert(result, IsNil)
	c.Assert(err, NotNil)
	observedTxInVote := NewObservedTxVoter(tx.ID, []ObservedTx{NewObservedTx(tx, s.ctx.BlockHeight(), GetRandomPubKey())})
	s.k.SetObservedTxInVoter(s.ctx, observedTxInVote)
	result, err = s.querier(s.ctx, []string{query.QueryTxInVoter.Key, tx.ID.String()}, req)
	c.Assert(err, IsNil)
	c.Assert(result, NotNil)
	var voter ObservedTxVoter
	c.Assert(s.k.Cdc().UnmarshalJSON(result, &voter), IsNil)
	c.Assert(voter.Valid(), IsNil)
}

func (s *QuerierSuite) TestQueryTx(c *C) {
	req := abci.RequestQuery{
		Data:   nil,
		Path:   query.QueryTxIn.Key,
		Height: s.ctx.BlockHeight(),
		Prove:  false,
	}
	tx := GetRandomTx()
	// test get tx in
	result, err := s.querier(s.ctx, []string{query.QueryTxIn.Key, tx.ID.String()}, req)
	c.Assert(result, IsNil)
	c.Assert(err, NotNil)
	nodeAccount := GetRandomNodeAccount(NodeActive)
	s.k.SetNodeAccount(s.ctx, nodeAccount)
	voter, err := s.k.GetObservedTxInVoter(s.ctx, tx.ID)
	c.Assert(err, IsNil)
	voter.Add(NewObservedTx(tx, s.ctx.BlockHeight(), nodeAccount.PubKeySet.Secp256k1), nodeAccount.NodeAddress)
	s.k.SetObservedTxInVoter(s.ctx, voter)
	result, err = s.querier(s.ctx, []string{query.QueryTxIn.Key, tx.ID.String()}, req)
	c.Assert(err, IsNil)
	var newTx ObservedTx
	c.Assert(s.k.Cdc().UnmarshalJSON(result, &newTx), IsNil)
	c.Assert(newTx.Valid(), IsNil)
}

func (s *QuerierSuite) TestQueryKeyGen(c *C) {
	req := abci.RequestQuery{
		Data:   nil,
		Path:   query.QueryKeygensPubkey.Key,
		Height: s.ctx.BlockHeight(),
		Prove:  false,
	}

	result, err := s.querier(s.ctx, []string{
		query.QueryKeygensPubkey.Key,
		"whatever",
	}, req)

	c.Assert(result, IsNil)
	c.Assert(err, NotNil)

	result, err = s.querier(s.ctx, []string{
		query.QueryKeygensPubkey.Key,
		"10000",
	}, req)

	c.Assert(result, IsNil)
	c.Assert(err, NotNil)

	result, err = s.querier(s.ctx, []string{
		query.QueryKeygensPubkey.Key,
		strconv.FormatInt(s.ctx.BlockHeight(), 10),
	}, req)
	c.Assert(result, NotNil)
	c.Assert(err, IsNil)

	result, err = s.querier(s.ctx, []string{
		query.QueryKeygensPubkey.Key,
		strconv.FormatInt(s.ctx.BlockHeight(), 10),
		GetRandomPubKey().String(),
	}, req)
	c.Assert(result, NotNil)
	c.Assert(err, IsNil)
}

func (s *QuerierSuite) TestQueryQueue(c *C) {
	result, err := s.querier(s.ctx, []string{
		query.QueryQueue.Key,
		strconv.FormatInt(s.ctx.BlockHeight(), 10),
	}, abci.RequestQuery{})
	c.Assert(result, NotNil)
	c.Assert(err, IsNil)
	var q QueryQueue
	c.Assert(s.k.Cdc().UnmarshalJSON(result, &q), IsNil)
}

func (s *QuerierSuite) TestQueryHeights(c *C) {
	result, err := s.querier(s.ctx, []string{
		query.QueryHeights.Key,
		strconv.FormatInt(s.ctx.BlockHeight(), 10),
	}, abci.RequestQuery{})
	c.Assert(result, IsNil)
	c.Assert(err, NotNil)

	result, err = s.querier(s.ctx, []string{
		query.QueryHeights.Key,
	}, abci.RequestQuery{})
	c.Assert(result, NotNil)
	c.Assert(err, IsNil)
	var q QueryResHeights
	c.Assert(s.k.Cdc().UnmarshalJSON(result, &q), IsNil)

	result, err = s.querier(s.ctx, []string{
		query.QueryHeights.Key,
		"BTC",
	}, abci.RequestQuery{})
	c.Assert(result, NotNil)
	c.Assert(err, IsNil)
	c.Assert(s.k.Cdc().UnmarshalJSON(result, &q), IsNil)

	result, err = s.querier(s.ctx, []string{
		query.QueryChainHeights.Key,
		"BTC",
	}, abci.RequestQuery{})
	c.Assert(result, NotNil)
	c.Assert(err, IsNil)
	c.Assert(s.k.Cdc().UnmarshalJSON(result, &q), IsNil)
}

func (s *QuerierSuite) TestQueryObservers(c *C) {
	result, err := s.querier(s.ctx, []string{
		query.QueryObservers.Key,
	}, abci.RequestQuery{})
	c.Assert(result, NotNil)
	c.Assert(err, IsNil)
	var q []string
	c.Assert(s.k.Cdc().UnmarshalJSON(result, &q), IsNil)

	result, err = s.querier(s.ctx, []string{
		query.QueryObserver.Key,
		"whatever",
	}, abci.RequestQuery{})
	c.Assert(result, IsNil)
	c.Assert(err, NotNil)

	activeNode := GetRandomNodeAccount(NodeActive)
	s.k.SetNodeAccount(s.ctx, activeNode)
	s.k.SetNodeAccount(s.ctx, GetRandomNodeAccount(NodeActive))
	result, err = s.querier(s.ctx, []string{
		query.QueryObserver.Key,
		GetRandomBech32Addr().String(),
	}, abci.RequestQuery{})
	c.Assert(result, IsNil)
	c.Assert(err, NotNil)

	result, err = s.querier(s.ctx, []string{
		query.QueryObserver.Key,
		activeNode.NodeAddress.String(),
	}, abci.RequestQuery{})
	c.Assert(result, NotNil)
	c.Assert(err, IsNil)
	var na NodeAccount
	c.Assert(s.k.Cdc().UnmarshalJSON(result, &na), IsNil)
}

func (s *QuerierSuite) TestQueryTssSigners(c *C) {
	result, err := s.querier(s.ctx, []string{
		query.QueryTSSSigners.Key,
		"",
	}, abci.RequestQuery{})
	c.Assert(result, IsNil)
	c.Assert(err, NotNil)

	result, err = s.querier(s.ctx, []string{
		query.QueryTSSSigners.Key,
		"blabalbal",
	}, abci.RequestQuery{})
	c.Assert(result, IsNil)
	c.Assert(err, NotNil)

	result, err = s.querier(s.ctx, []string{
		query.QueryTSSSigners.Key,
		GetRandomVault().PubKey.String(),
	}, abci.RequestQuery{})
	c.Assert(result, IsNil)
	c.Assert(err, NotNil)
	vault := GetRandomVault()
	vault.Membership = common.PubKeys{
		GetRandomPubKey(),
		GetRandomPubKey(),
		GetRandomPubKey(),
		GetRandomPubKey(),
		GetRandomPubKey(),
	}
	s.k.SetVault(s.ctx, vault)
	result, err = s.querier(s.ctx, []string{
		query.QueryTSSSigners.Key,
		vault.PubKey.String(),
	}, abci.RequestQuery{})
	c.Assert(result, NotNil)
	c.Assert(err, IsNil)

	var signerParty common.PubKeys
	c.Assert(s.k.Cdc().UnmarshalJSON(result, &signerParty), IsNil)
}

func (s *QuerierSuite) TestQueryConstantValues(c *C) {
	result, err := s.querier(s.ctx, []string{
		query.QueryConstantValues.Key,
	}, abci.RequestQuery{})
	c.Assert(result, NotNil)
	c.Assert(err, IsNil)
}

func (s *QuerierSuite) TestQueryMimir(c *C) {
	s.k.SetMimir(s.ctx, "hello", 111)
	result, err := s.querier(s.ctx, []string{
		query.QueryMimirValues.Key,
	}, abci.RequestQuery{})
	c.Assert(result, NotNil)
	c.Assert(err, IsNil)
	var m map[string]int64
	c.Assert(s.k.Cdc().UnmarshalJSON(result, &m), IsNil)
}

func (s *QuerierSuite) TestQueryBan(c *C) {
	result, err := s.querier(s.ctx, []string{
		query.QueryBan.Key,
	}, abci.RequestQuery{})
	c.Assert(result, IsNil)
	c.Assert(err, NotNil)

	result, err = s.querier(s.ctx, []string{
		query.QueryBan.Key,
		"Whatever",
	}, abci.RequestQuery{})
	c.Assert(result, IsNil)
	c.Assert(err, NotNil)

	result, err = s.querier(s.ctx, []string{
		query.QueryBan.Key,
		GetRandomBech32Addr().String(),
	}, abci.RequestQuery{})
	c.Assert(result, NotNil)
	c.Assert(err, IsNil)
}

func (s *QuerierSuite) TestQueryNodeAccount(c *C) {
	result, err := s.querier(s.ctx, []string{
		query.QueryNodeAccount.Key,
	}, abci.RequestQuery{})
	c.Assert(result, IsNil)
	c.Assert(err, NotNil)

	result, err = s.querier(s.ctx, []string{
		query.QueryNodeAccount.Key,
		"Whatever",
	}, abci.RequestQuery{})
	c.Assert(result, IsNil)
	c.Assert(err, NotNil)

	na := GetRandomNodeAccount(NodeActive)
	s.k.SetNodeAccount(s.ctx, na)
	result, err = s.querier(s.ctx, []string{
		query.QueryNodeAccount.Key,
		na.NodeAddress.String(),
	}, abci.RequestQuery{})
	c.Assert(result, NotNil)
	c.Assert(err, IsNil)
	var r QueryNodeAccount
	c.Assert(s.k.Cdc().UnmarshalJSON(result, &r), IsNil)
}

func (s *QuerierSuite) TestQueryNodeAccountCheck(c *C) {
	result, err := s.querier(s.ctx, []string{
		query.QueryNodeAccountCheck.Key,
	}, abci.RequestQuery{})
	c.Assert(result, IsNil)
	c.Assert(err, NotNil)

	result, err = s.querier(s.ctx, []string{
		query.QueryNodeAccountCheck.Key,
		"Whatever",
	}, abci.RequestQuery{})
	c.Assert(result, IsNil)
	c.Assert(err, NotNil)

	na := GetRandomNodeAccount(NodeStandby)
	s.k.SetNodeAccount(s.ctx, na)
	result, err = s.querier(s.ctx, []string{
		query.QueryNodeAccountCheck.Key,
		na.NodeAddress.String(),
	}, abci.RequestQuery{})
	c.Assert(result, NotNil)
	c.Assert(err, IsNil)
	var r QueryNodeAccountPreflightCheck
	c.Assert(s.k.Cdc().UnmarshalJSON(result, &r), IsNil)
}

func (s *QuerierSuite) TestQueryPoolAddresses(c *C) {
	na := GetRandomNodeAccount(NodeActive)
	s.k.SetNodeAccount(s.ctx, na)
	result, err := s.querier(s.ctx, []string{
		query.QueryPoolAddresses.Key,
		na.NodeAddress.String(),
	}, abci.RequestQuery{})
	c.Assert(result, NotNil)
	c.Assert(err, IsNil)

	var resp struct {
		Current []struct {
			Chain   common.Chain   `json:"chain"`
			PubKey  common.PubKey  `json:"pub_key"`
			Address common.Address `json:"address"`
			Halted  bool           `json:"halted"`
		} `json:"current"`
	}
	c.Assert(s.k.Cdc().UnmarshalJSON(result, &resp), IsNil)
}

func (s *QuerierSuite) TestQueryKeysignArrayPubKey(c *C) {
	na := GetRandomNodeAccount(NodeActive)
	s.k.SetNodeAccount(s.ctx, na)
	result, err := s.querier(s.ctx, []string{
		query.QueryKeysignArrayPubkey.Key,
	}, abci.RequestQuery{})
	c.Assert(result, IsNil)
	c.Assert(err, NotNil)

	result, err = s.querier(s.ctx, []string{
		query.QueryKeysignArrayPubkey.Key,
		"asdf",
	}, abci.RequestQuery{})
	c.Assert(result, IsNil)
	c.Assert(err, NotNil)

	result, err = s.querier(s.ctx, []string{
		query.QueryKeysignArrayPubkey.Key,
		strconv.FormatInt(s.ctx.BlockHeight(), 10),
	}, abci.RequestQuery{})
	c.Assert(result, NotNil)
	c.Assert(err, IsNil)
	var r QueryKeysign
	c.Assert(s.k.Cdc().UnmarshalJSON(result, &r), IsNil)
}

func (s *QuerierSuite) TestQueryVaultData(c *C) {
	result, err := s.querier(s.ctx, []string{
		query.QueryVaultData.Key,
	}, abci.RequestQuery{})
	c.Assert(result, NotNil)
	c.Assert(err, IsNil)
	var r VaultData
	c.Assert(s.k.Cdc().UnmarshalJSON(result, &r), IsNil)
}

func (s *QuerierSuite) TestQueryAsgardVault(c *C) {
	s.k.SetVault(s.ctx, GetRandomVault())
	result, err := s.querier(s.ctx, []string{
		query.QueryVaultsAsgard.Key,
	}, abci.RequestQuery{})
	c.Assert(result, NotNil)
	c.Assert(err, IsNil)
	var r Vaults
	c.Assert(s.k.Cdc().UnmarshalJSON(result, &r), IsNil)
}

func (s *QuerierSuite) TestQueryYggdrasilVault(c *C) {
	vault := GetRandomVault()
	vault.Type = YggdrasilVault
	vault.AddFunds(common.Coins{
		common.NewCoin(common.BNBAsset, cosmos.NewUint(common.One*100)),
	})
	s.k.SetVault(s.ctx, vault)
	result, err := s.querier(s.ctx, []string{
		query.QueryVaultsYggdrasil.Key,
	}, abci.RequestQuery{})
	c.Assert(result, NotNil)
	c.Assert(err, IsNil)
	var r []QueryYggdrasilVaults
	c.Assert(s.k.Cdc().UnmarshalJSON(result, &r), IsNil)
}

func (s *QuerierSuite) TestQueryVaultPubKeys(c *C) {
	vault := GetRandomVault()
	vault.Type = YggdrasilVault
	vault.AddFunds(common.Coins{
		common.NewCoin(common.BNBAsset, cosmos.NewUint(common.One*100)),
	})
	s.k.SetVault(s.ctx, vault)
	result, err := s.querier(s.ctx, []string{
		query.QueryVaultPubkeys.Key,
	}, abci.RequestQuery{})
	c.Assert(result, NotNil)
	c.Assert(err, IsNil)
	var r struct {
		Asgard    common.PubKeys `json:"asgard"`
		Yggdrasil common.PubKeys `json:"yggdrasil"`
	}
	c.Assert(s.k.Cdc().UnmarshalJSON(result, &r), IsNil)
}

func (s *QuerierSuite) TestQueryBalanceModule(c *C) {
	s.k.SetVault(s.ctx, GetRandomVault())
	result, err := s.querier(s.ctx, []string{
		query.QueryBalanceModule.Key,
	}, abci.RequestQuery{})
	c.Assert(result, NotNil)
	c.Assert(err, IsNil)
	var r sdk.Coins
	c.Assert(s.k.Cdc().UnmarshalJSON(result, &r), IsNil)
}

func (s *QuerierSuite) TestQueryVault(c *C) {
	vault := GetRandomVault()

	// Not enough argument
	result, err := s.querier(s.ctx, []string{
		query.QueryVault.Key,
		"BNB",
	}, abci.RequestQuery{})

	c.Assert(result, IsNil)
	c.Assert(err, NotNil)

	// invalid chain
	result, err = s.querier(s.ctx, []string{
		query.QueryVault.Key,
		"A",
		GetRandomBNBAddress().String(),
	}, abci.RequestQuery{})

	c.Assert(result, IsNil)
	c.Assert(err, NotNil)

	// invalid address
	result, err = s.querier(s.ctx, []string{
		query.QueryVault.Key,
		"BTC",
		GetRandomBNBAddress().String(),
	}, abci.RequestQuery{})

	c.Assert(result, IsNil)
	c.Assert(err, NotNil)

	s.k.SetVault(s.ctx, vault)
	addr, err := vault.PubKey.GetAddress(common.BNBChain)
	c.Assert(err, IsNil)
	result, err = s.querier(s.ctx, []string{
		query.QueryVault.Key,
		"BNB",
		addr.String(),
	}, abci.RequestQuery{})
	c.Assert(err, IsNil)
	c.Assert(result, NotNil)
	var returnVault Vault
	c.Assert(s.k.Cdc().UnmarshalJSON(result, &returnVault), IsNil)
	c.Assert(vault.PubKey.Equals(returnVault.PubKey), Equals, true)
	c.Assert(vault.Type, Equals, returnVault.Type)
	c.Assert(vault.Status, Equals, returnVault.Status)
	c.Assert(vault.BlockHeight, Equals, returnVault.BlockHeight)
}
