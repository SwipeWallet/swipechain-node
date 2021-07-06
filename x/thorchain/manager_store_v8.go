package thorchain

import (
	"fmt"

	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/common/cosmos"
	"gitlab.com/thorchain/thornode/constants"
	"gitlab.com/thorchain/thornode/x/thorchain/keeper"
)

func getPendingTxOut(ctx cosmos.Context, k keeper.Keeper, constAccessor constants.ConstantValues) ([]*TxOutItem, error) {
	signingTransactionPeriod := constAccessor.GetInt64Value(constants.SigningTransactionPeriod)
	startHeight := common.BlockHeight(ctx) - signingTransactionPeriod
	var txOutItems []*TxOutItem
	for height := startHeight; height <= common.BlockHeight(ctx); height++ {
		txs, err := k.GetTxOut(ctx, height)
		if err != nil {
			ctx.Logger().Error("fail to get tx out array from key value store", "error", err)
			return nil, fmt.Errorf("fail to get tx out array from key value store: %w", err)
		}
		for _, tx := range txs.TxArray {
			// migration , those internal triggered txout has blank in txhash
			if tx.OutHash.IsEmpty() && !tx.InHash.Equals(common.BlankTxID) {
				txOutItems = append(txOutItems, tx)
			}
		}
	}
	return txOutItems, nil
}

func getPoolAssetBalance(asset common.Asset, vaults Vaults, txOutItems []*TxOutItem) cosmos.Uint {
	amount := cosmos.ZeroUint()
	for _, v := range vaults {
		amount = amount.Add(v.GetCoin(asset).Amount)
	}
	for _, item := range txOutItems {
		if item.Coin.Asset.Equals(asset) {
			amount = common.SafeSub(amount, item.Coin.Amount)
		}
	}

	return amount
}

func fixPoolAsset(ctx cosmos.Context, keep keeper.Keeper, constAccessor constants.ConstantValues) error {
	pools, err := keep.GetPools(ctx)
	if err != nil {
		return fmt.Errorf("fail to get pools: %w", err)
	}
	txOutItems, err := getPendingTxOut(ctx, keep, constAccessor)
	if err != nil {
		return fmt.Errorf("fail to get txout items: %w", err)
	}

	vaults := Vaults{}
	iter := keep.GetVaultIterator(ctx)
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		var vault Vault
		if err := keep.Cdc().UnmarshalBinaryBare(iter.Value(), &vault); err != nil {
			ctx.Logger().Error("fail to unmarshal vault", "error", err)
			continue
		}
		if vault.IsEmpty() {
			continue
		}
		if !vault.HasFunds() {
			continue
		}

		// this include yggdrasil vaults as well
		if vault.Status != ActiveVault && vault.Status != RetiringVault {
			continue
		}
		vaults = append(vaults, vault)
	}

	for _, p := range pools {
		totalAsset := getPoolAssetBalance(p.Asset, vaults, txOutItems)
		ctx.Logger().Info("update pool asset amount", "before", p.BalanceAsset.String(), "after", totalAsset.String())
		p.BalanceAsset = totalAsset
		if err := keep.SetPool(ctx, p); err != nil {
			ctx.Logger().Error("fail to save pool: %w", err)
		}
	}
	return nil
}
