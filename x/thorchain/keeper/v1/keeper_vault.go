package keeperv1

import (
	"errors"
	"fmt"

	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/common/cosmos"
	kvTypes "gitlab.com/thorchain/thornode/x/thorchain/keeper/types"
)

// GetVaultIterator only iterate vault pools
func (k KVStore) GetVaultIterator(ctx cosmos.Context) cosmos.Iterator {
	return k.getIterator(ctx, prefixVault)
}

// SetVault save the Vault object to store
func (k KVStore) SetVault(ctx cosmos.Context, vault Vault) error {
	if vault.IsAsgard() {
		if err := k.addAsgardIndex(ctx, vault.PubKey); err != nil {
			return err
		}
	}

	k.set(ctx, k.GetKey(ctx, prefixVault, vault.PubKey.String()), vault)
	return nil
}

// VaultExists check whether the given pubkey is associated with a vault
func (k KVStore) VaultExists(ctx cosmos.Context, pk common.PubKey) bool {
	return k.has(ctx, k.GetKey(ctx, prefixVault, pk.String()))
}

// GetVault get Vault with the given pubkey from data store
func (k KVStore) GetVault(ctx cosmos.Context, pk common.PubKey) (Vault, error) {
	record := Vault{
		BlockHeight: common.BlockHeight(ctx),
		PubKey:      pk,
	}
	ok, err := k.get(ctx, k.GetKey(ctx, prefixVault, pk.String()), &record)
	if !ok {
		return record, fmt.Errorf("vault with pubkey(%s) doesn't exist: %w", pk, kvTypes.ErrVaultNotFound)
	}
	if record.PubKey.IsEmpty() {
		record.PubKey = pk
	}
	return record, err
}

// HasValidVaultPools check the data store to see whether we have a valid vault
func (k KVStore) HasValidVaultPools(ctx cosmos.Context) (bool, error) {
	iterator := k.GetVaultIterator(ctx)
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		var vault Vault
		if err := k.cdc.UnmarshalBinaryBare(iterator.Value(), &vault); err != nil {
			return false, dbError(ctx, "fail to unmarshal vault", err)
		}
		if vault.HasFunds() {
			return true, nil
		}
	}
	return false, nil
}

func (k KVStore) getAsgardIndex(ctx cosmos.Context) (common.PubKeys, error) {
	record := make(common.PubKeys, 0)
	_, err := k.get(ctx, k.GetKey(ctx, prefixVaultAsgardIndex, ""), &record)
	return record, err
}

func (k KVStore) addAsgardIndex(ctx cosmos.Context, pubkey common.PubKey) error {
	pks, err := k.getAsgardIndex(ctx)
	if err != nil {
		return err
	}
	for _, pk := range pks {
		if pk.Equals(pubkey) {
			return nil
		}
	}
	pks = append(pks, pubkey)
	k.set(ctx, k.GetKey(ctx, prefixVaultAsgardIndex, ""), pks)
	return nil
}

// GetAsgardVaults return all asgard vaults
func (k KVStore) GetAsgardVaults(ctx cosmos.Context) (Vaults, error) {
	pks, err := k.getAsgardIndex(ctx)
	if err != nil {
		return nil, err
	}

	var asgards Vaults
	for _, pk := range pks {
		vault, err := k.GetVault(ctx, pk)
		if err != nil {
			return nil, err
		}
		if vault.IsAsgard() {
			asgards = append(asgards, vault)
		}
	}

	return asgards, nil
}

// GetAsgardVaultsByStatus get all the asgard vault that have the given status
func (k KVStore) GetAsgardVaultsByStatus(ctx cosmos.Context, status VaultStatus) (Vaults, error) {
	all, err := k.GetAsgardVaults(ctx)
	if err != nil {
		return nil, err
	}

	var asgards Vaults
	for _, vault := range all {
		if vault.Status == status {
			asgards = append(asgards, vault)
		}
	}

	return asgards, nil
}

// DeleteVault remove the given vault from data store
func (k KVStore) DeleteVault(ctx cosmos.Context, pubkey common.PubKey) error {
	vault, err := k.GetVault(ctx, pubkey)
	if err != nil {
		if errors.Is(err, kvTypes.ErrVaultNotFound) {
			return nil
		}
		return err
	}

	if vault.HasFunds() {
		return errors.New("unable to delete vault: it still contains funds")
	}

	if vault.IsAsgard() {
		pks, err := k.getAsgardIndex(ctx)
		if err != nil {
			return err
		}

		newPks := common.PubKeys{}
		for _, pk := range pks {
			if !pk.Equals(pubkey) {
				newPks = append(newPks, pk)
			}
		}

		k.set(ctx, k.GetKey(ctx, prefixVaultAsgardIndex, ""), newPks)
	}
	// delete the actual vault
	k.del(ctx, k.GetKey(ctx, prefixVault, vault.PubKey.String()))
	return nil
}
