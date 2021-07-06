package keeper

import (
	"gitlab.com/thorchain/thornode/x/thorchain/types"
)

const (
	ModuleName  = types.ModuleName
	ReserveName = types.ReserveName
	AsgardName  = types.AsgardName
	BondName    = types.BondName
	RouterKey   = types.RouterKey
	StoreKey    = types.StoreKey

	ActiveVault = types.ActiveVault

	// Node status
	NodeActive = types.Active
)

var (
	NewPool              = types.NewPool
	NewJail              = types.NewJail
	ModuleCdc            = types.ModuleCdc
	RegisterCodec        = types.RegisterCodec
	GetRandomVault       = types.GetRandomVault
	GetRandomNodeAccount = types.GetRandomNodeAccount
	GetRandomBNBAddress  = types.GetRandomBNBAddress
	GetRandomTxHash      = types.GetRandomTxHash
	GetRandomBech32Addr  = types.GetRandomBech32Addr
	GetRandomPubKey      = types.GetRandomPubKey
)

type (
	MsgSwap = types.MsgSwap

	PoolStatus              = types.PoolStatus
	Pool                    = types.Pool
	Pools                   = types.Pools
	Staker                  = types.Staker
	ObservedTxVoter         = types.ObservedTxVoter
	BanVoter                = types.BanVoter
	ErrataTxVoter           = types.ErrataTxVoter
	TssVoter                = types.TssVoter
	TssKeysignFailVoter     = types.TssKeysignFailVoter
	TxOutItem               = types.TxOutItem
	TxOut                   = types.TxOut
	KeygenBlock             = types.KeygenBlock
	ReserveContributors     = types.ReserveContributors
	Vault                   = types.Vault
	Vaults                  = types.Vaults
	Jail                    = types.Jail
	NodeAccount             = types.NodeAccount
	NodeAccounts            = types.NodeAccounts
	NodeStatus              = types.NodeStatus
	VaultData               = types.VaultData
	VaultStatus             = types.VaultStatus
	TxMarker                = types.TxMarker
	TxMarkers               = types.TxMarkers
	NetworkFee              = types.NetworkFee
	ObservedNetworkFeeVoter = types.ObservedNetworkFeeVoter
	RagnarokUnstakePosition = types.RagnarokUnstakePosition
)
