package keeperv1

import (
	"github.com/cosmos/cosmos-sdk/types"
	. "gopkg.in/check.v1"

	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/common/cosmos"
)

type KeeperVaultSuite struct{}

var _ = Suite(&KeeperVaultSuite{})

func (s *KeeperVaultSuite) TestVault(c *C) {
	ctx, k := setupKeeperForTest(c)
	existVault, err := k.HasValidVaultPools(ctx)
	c.Check(err, IsNil)
	c.Check(existVault, Equals, false)

	pubKey := GetRandomPubKey()
	yggdrasil := NewVault(common.BlockHeight(ctx), ActiveVault, YggdrasilVault, pubKey, common.Chains{common.BNBChain})
	err = k.SetVault(ctx, yggdrasil)
	c.Assert(err, IsNil)
	c.Assert(k.VaultExists(ctx, pubKey), Equals, true)
	pubKey1 := GetRandomPubKey()
	yggdrasil1 := NewVault(common.BlockHeight(ctx), ActiveVault, YggdrasilVault, pubKey1, common.Chains{common.BNBChain})
	yggdrasil1.PendingTxBlockHeights = []int64{35}
	yggdrasil1.Coins = common.Coins{
		common.NewCoin(common.BNBAsset, types.NewUint(100)),
	}
	c.Assert(k.SetVault(ctx, yggdrasil1), IsNil)
	ygg, err := k.GetVault(ctx, pubKey1)
	c.Assert(err, IsNil)
	c.Assert(ygg.IsEmpty(), Equals, false)
	c.Assert(ygg.PendingTxBlockHeights, HasLen, 1)
	c.Assert(ygg.PendingTxBlockHeights[0], Equals, int64(35))
	hasYgg, err := k.HasValidVaultPools(ctx)
	c.Assert(err, IsNil)
	c.Assert(hasYgg, Equals, true)

	asgards, err := k.GetAsgardVaultsByStatus(ctx, ActiveVault)
	c.Assert(err, IsNil)
	c.Assert(asgards, HasLen, 0)
	pubKey = GetRandomPubKey()
	asgard := NewVault(common.BlockHeight(ctx), ActiveVault, AsgardVault, pubKey, common.Chains{common.BNBChain})
	c.Assert(k.SetVault(ctx, asgard), IsNil)
	asgard2 := NewVault(common.BlockHeight(ctx), InactiveVault, AsgardVault, GetRandomPubKey(), common.Chains{common.BNBChain})
	c.Assert(k.SetVault(ctx, asgard2), IsNil)
	asgards, err = k.GetAsgardVaultsByStatus(ctx, ActiveVault)
	c.Assert(err, IsNil)
	c.Assert(asgards, HasLen, 1)
	c.Check(asgards[0].PubKey.Equals(pubKey), Equals, true)

	c.Assert(k.DeleteVault(ctx, pubKey), IsNil)
	c.Assert(k.DeleteVault(ctx, pubKey), IsNil) // second time should also not error
	asgards, err = k.GetAsgardVaultsByStatus(ctx, ActiveVault)
	c.Assert(err, IsNil)
	c.Assert(asgards, HasLen, 0)

	vault1 := NewVault(1024, ActiveVault, AsgardVault, GetRandomPubKey(), common.Chains{common.BNBChain})
	vault1.AddFunds(common.Coins{
		common.NewCoin(common.BNBAsset, cosmos.NewUint(common.One*100)),
	})
	c.Check(k.SetVault(ctx, vault1), IsNil)
	c.Check(k.DeleteVault(ctx, vault1.PubKey), NotNil)
}
