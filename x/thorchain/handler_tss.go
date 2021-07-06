package thorchain

import (
	"fmt"

	"github.com/blang/semver"

	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/common/cosmos"
	"gitlab.com/thorchain/thornode/constants"
	"gitlab.com/thorchain/thornode/x/thorchain/keeper"
)

// TssHandler handle MsgTssPool
type TssHandler struct {
	keeper keeper.Keeper
	mgr    Manager
}

// NewTssHandler create a new handler to process MsgTssPool
func NewTssHandler(keeper keeper.Keeper, mgr Manager) TssHandler {
	return TssHandler{
		keeper: keeper,
		mgr:    mgr,
	}
}

// Run is the main entry for TssHandler
func (h TssHandler) Run(ctx cosmos.Context, m cosmos.Msg, version semver.Version, constAccessor constants.ConstantValues) (*cosmos.Result, error) {
	msg, ok := m.(MsgTssPool)
	if !ok {
		return nil, errInvalidMessage
	}
	err := h.validate(ctx, msg, version)
	if err != nil {
		ctx.Logger().Error("msg_tss_pool failed validation", "error", err)
		return nil, err
	}
	result, err := h.handle(ctx, msg, version, constAccessor)
	if err != nil {
		ctx.Logger().Error("failed to process MsgTssPool", "error", err)
		return nil, err
	}
	return result, err
}

func (h TssHandler) validate(ctx cosmos.Context, msg MsgTssPool, version semver.Version) error {
	if version.GTE(semver.MustParse("0.1.0")) {
		return h.validateV1(ctx, msg)
	}
	return errBadVersion
}

func (h TssHandler) validateV1(ctx cosmos.Context, msg MsgTssPool) error {
	if err := msg.ValidateBasic(); err != nil {
		return err
	}

	keygenBlock, err := h.keeper.GetKeygenBlock(ctx, msg.Height)
	if err != nil {
		return fmt.Errorf("fail to get keygen block from data store: %w", err)
	}

	for _, keygen := range keygenBlock.Keygens {
		for _, member := range keygen.Members {
			addr, err := member.GetThorAddress()
			if addr.Equals(msg.Signer) && err == nil {
				return nil
			}
		}
	}

	return cosmos.ErrUnauthorized("not authorized")
}

func (h TssHandler) handle(ctx cosmos.Context, msg MsgTssPool, version semver.Version, constAccessor constants.ConstantValues) (*cosmos.Result, error) {
	ctx.Logger().Info("handleMsgTssPool request", "ID:", msg.ID)
	if version.GTE(semver.MustParse("0.13.0")) {
		return h.handleV13(ctx, msg, version, constAccessor)
	} else if version.GTE(semver.MustParse("0.1.0")) {
		return h.handleV1(ctx, msg, version, constAccessor)
	}
	return nil, errBadVersion
}

func (h TssHandler) handleV1(ctx cosmos.Context, msg MsgTssPool, version semver.Version, constAccessor constants.ConstantValues) (*cosmos.Result, error) {
	ctx.Logger().Info(fmt.Sprintf("current version: %s", version.String()))
	if !msg.Blame.IsEmpty() {
		ctx.Logger().Error(msg.Blame.String())
	}

	voter, err := h.keeper.GetTssVoter(ctx, msg.ID)
	if err != nil {
		return nil, fmt.Errorf("fail to get tss voter: %w", err)
	}

	// when PoolPubKey is empty , which means TssVoter with id(msg.ID) doesn't
	// exist before, this is the first time to create it
	// set the PoolPubKey to the one in msg, there is no reason voter.PubKeys
	// have anything in it either, thus override it with msg.PubKeys as well
	if voter.PoolPubKey.IsEmpty() {
		voter.PoolPubKey = msg.PoolPubKey
		voter.PubKeys = msg.PubKeys
	}
	observeSlashPoints := constAccessor.GetInt64Value(constants.ObserveSlashPoints)
	observeFlex := constAccessor.GetInt64Value(constants.ObserveFlex)
	h.mgr.Slasher().IncSlashPoints(ctx, observeSlashPoints, msg.Signer)
	if !voter.Sign(msg.Signer, msg.Chains) {
		ctx.Logger().Info("signer already signed MsgTssPool", "signer", msg.Signer.String(), "txid", msg.ID)
		return &cosmos.Result{}, nil

	}
	h.keeper.SetTssVoter(ctx, voter)
	// doesn't have consensus yet
	if !voter.HasConsensus() {
		ctx.Logger().Info("not having consensus yet, return")
		return &cosmos.Result{}, nil
	}

	if voter.BlockHeight == 0 {
		voter.BlockHeight = common.BlockHeight(ctx)
		h.keeper.SetTssVoter(ctx, voter)
		h.mgr.Slasher().DecSlashPoints(ctx, observeSlashPoints, voter.Signers...)
		if msg.IsSuccess() {
			vaultType := YggdrasilVault
			if msg.KeygenType == AsgardKeygen {
				vaultType = AsgardVault
			}
			vault := NewVault(common.BlockHeight(ctx), ActiveVault, vaultType, voter.PoolPubKey, voter.ConsensusChains())
			vault.Membership = voter.PubKeys

			signingParty, err := ChooseSignerParty(voter.PubKeys, common.BlockHeight(ctx), len(voter.PubKeys))
			if err != nil {
				return nil, fmt.Errorf("fail to choose signing party: %w", err)
			}
			vault.SigningParty = signingParty
			if err := h.keeper.SetVault(ctx, vault); err != nil {
				return nil, fmt.Errorf("fail to save vault: %w", err)
			}
			if err := h.mgr.VaultMgr().RotateVault(ctx, vault); err != nil {
				return nil, fmt.Errorf("fail to rotate vault: %w", err)
			}
		} else {
			// if a node fail to join the keygen, thus hold off the network
			// from churning then it will be slashed accordingly
			slashPoints := constAccessor.GetInt64Value(constants.FailKeygenSlashPoints)
			for _, node := range msg.Blame.BlameNodes {
				nodePubKey, err := common.NewPubKey(node.Pubkey)
				if err != nil {
					return nil, ErrInternal(err, fmt.Sprintf("fail to parse pubkey(%s)", node.Pubkey))
				}

				na, err := h.keeper.GetNodeAccountByPubKey(ctx, nodePubKey)
				if err != nil {
					return nil, fmt.Errorf("fail to get node from it's pub key: %w", err)
				}
				if na.Status == NodeActive {
					if err := h.keeper.IncNodeAccountSlashPoints(ctx, na.NodeAddress, slashPoints); err != nil {
						ctx.Logger().Error("fail to inc slash points", "error", err)
					}
				} else {
					// go to jail
					jailTime := constAccessor.GetInt64Value(constants.JailTimeKeygen)
					releaseHeight := common.BlockHeight(ctx) + jailTime
					reason := "failed to perform keygen"
					if err := h.keeper.SetNodeAccountJail(ctx, na.NodeAddress, releaseHeight, reason); err != nil {
						ctx.Logger().Error("fail to set node account jail", "node address", na.NodeAddress, "reason", reason, "error", err)
					}

					// take out bond from the node account and add it to vault bond reward RUNE
					// thus good behaviour node will get reward
					reserveVault, err := h.keeper.GetVaultData(ctx)
					if err != nil {
						return nil, fmt.Errorf("fail to get reserve vault: %w", err)
					}

					slashBond := reserveVault.CalcNodeRewards(cosmos.NewUint(uint64(slashPoints)))
					if slashBond.GT(na.Bond) {
						slashBond = na.Bond
					}
					ctx.Logger().Info("fail keygen , slash bond", "address", na.NodeAddress, "amount", slashBond.String())
					na.Bond = common.SafeSub(na.Bond, slashBond)
					if common.RuneAsset().Chain.Equals(common.THORChain) {
						coin := common.NewCoin(common.RuneNative, slashBond)
						if err := h.keeper.SendFromModuleToModule(ctx, BondName, ReserveName, coin); err != nil {
							return nil, fmt.Errorf("fail to transfer funds from bond to reserve: %w", err)
						}
					} else {
						reserveVault.TotalReserve = reserveVault.TotalReserve.Add(slashBond)
						if err := h.keeper.SetVaultData(ctx, reserveVault); err != nil {
							ctx.Logger().Error("fail to set vault data", "error", err)
						}
					}

				}
				if err := h.keeper.SetNodeAccount(ctx, na); err != nil {
					return nil, fmt.Errorf("fail to save node account: %w", err)
				}
			}

		}
		return &cosmos.Result{}, nil
	}

	if (voter.BlockHeight + observeFlex) >= common.BlockHeight(ctx) {
		h.mgr.Slasher().DecSlashPoints(ctx, observeSlashPoints, msg.Signer)
	}

	return &cosmos.Result{}, nil
}

func (h TssHandler) handleV13(ctx cosmos.Context, msg MsgTssPool, version semver.Version, constAccessor constants.ConstantValues) (*cosmos.Result, error) {
	ctx.Logger().Info(fmt.Sprintf("current version: %s", version.String()))
	if !msg.Blame.IsEmpty() {
		ctx.Logger().Error(msg.Blame.String())
	}

	voter, err := h.keeper.GetTssVoter(ctx, msg.ID)
	if err != nil {
		return nil, fmt.Errorf("fail to get tss voter: %w", err)
	}

	// when PoolPubKey is empty , which means TssVoter with id(msg.ID) doesn't
	// exist before, this is the first time to create it
	// set the PoolPubKey to the one in msg, there is no reason voter.PubKeys
	// have anything in it either, thus override it with msg.PubKeys as well
	if voter.PoolPubKey.IsEmpty() {
		voter.PoolPubKey = msg.PoolPubKey
		voter.PubKeys = msg.PubKeys
	}
	observeSlashPoints := constAccessor.GetInt64Value(constants.ObserveSlashPoints)
	observeFlex := constAccessor.GetInt64Value(constants.ObserveFlex)
	h.mgr.Slasher().IncSlashPoints(ctx, observeSlashPoints, msg.Signer)
	if !voter.Sign(msg.Signer, msg.Chains) {
		ctx.Logger().Info("signer already signed MsgTssPool", "signer", msg.Signer.String(), "txid", msg.ID)
		return &cosmos.Result{}, nil

	}
	h.keeper.SetTssVoter(ctx, voter)
	// doesn't have consensus yet
	if !voter.HasConsensusV13() {
		ctx.Logger().Info("not having consensus yet, return")
		return &cosmos.Result{}, nil
	}

	if voter.BlockHeight == 0 {
		voter.BlockHeight = common.BlockHeight(ctx)
		h.keeper.SetTssVoter(ctx, voter)
		h.mgr.Slasher().DecSlashPoints(ctx, observeSlashPoints, voter.Signers...)
		if msg.IsSuccess() {
			vaultType := YggdrasilVault
			if msg.KeygenType == AsgardKeygen {
				vaultType = AsgardVault
			}
			vault := NewVault(common.BlockHeight(ctx), ActiveVault, vaultType, voter.PoolPubKey, voter.ConsensusChainsV13())
			vault.Membership = voter.PubKeys

			signingParty, err := ChooseSignerParty(voter.PubKeys, common.BlockHeight(ctx), len(voter.PubKeys))
			if err != nil {
				return nil, fmt.Errorf("fail to choose signing party: %w", err)
			}
			vault.SigningParty = signingParty
			if err := h.keeper.SetVault(ctx, vault); err != nil {
				return nil, fmt.Errorf("fail to save vault: %w", err)
			}
			if err := h.mgr.VaultMgr().RotateVault(ctx, vault); err != nil {
				return nil, fmt.Errorf("fail to rotate vault: %w", err)
			}
		} else {
			// if a node fail to join the keygen, thus hold off the network
			// from churning then it will be slashed accordingly
			slashPoints := constAccessor.GetInt64Value(constants.FailKeygenSlashPoints)
			for _, node := range msg.Blame.BlameNodes {
				nodePubKey, err := common.NewPubKey(node.Pubkey)
				if err != nil {
					return nil, ErrInternal(err, fmt.Sprintf("fail to parse pubkey(%s)", node.Pubkey))
				}

				na, err := h.keeper.GetNodeAccountByPubKey(ctx, nodePubKey)
				if err != nil {
					return nil, fmt.Errorf("fail to get node from it's pub key: %w", err)
				}
				if na.Status == NodeActive {
					if err := h.keeper.IncNodeAccountSlashPoints(ctx, na.NodeAddress, slashPoints); err != nil {
						ctx.Logger().Error("fail to inc slash points", "error", err)
					}
				} else {
					// go to jail
					jailTime := constAccessor.GetInt64Value(constants.JailTimeKeygen)
					releaseHeight := common.BlockHeight(ctx) + jailTime
					reason := "failed to perform keygen"
					if err := h.keeper.SetNodeAccountJail(ctx, na.NodeAddress, releaseHeight, reason); err != nil {
						ctx.Logger().Error("fail to set node account jail", "node address", na.NodeAddress, "reason", reason, "error", err)
					}

					// take out bond from the node account and add it to vault bond reward RUNE
					// thus good behaviour node will get reward
					reserveVault, err := h.keeper.GetVaultData(ctx)
					if err != nil {
						return nil, fmt.Errorf("fail to get reserve vault: %w", err)
					}

					slashBond := reserveVault.CalcNodeRewards(cosmos.NewUint(uint64(slashPoints)))
					if slashBond.GT(na.Bond) {
						slashBond = na.Bond
					}
					ctx.Logger().Info("fail keygen , slash bond", "address", na.NodeAddress, "amount", slashBond.String())
					na.Bond = common.SafeSub(na.Bond, slashBond)
					if common.RuneAsset().Chain.Equals(common.THORChain) {
						coin := common.NewCoin(common.RuneNative, slashBond)
						if err := h.keeper.SendFromModuleToModule(ctx, BondName, ReserveName, coin); err != nil {
							return nil, fmt.Errorf("fail to transfer funds from bond to reserve: %w", err)
						}
					} else {
						reserveVault.TotalReserve = reserveVault.TotalReserve.Add(slashBond)
						if err := h.keeper.SetVaultData(ctx, reserveVault); err != nil {
							ctx.Logger().Error("fail to set vault data", "error", err)
						}
					}

				}
				if err := h.keeper.SetNodeAccount(ctx, na); err != nil {
					return nil, fmt.Errorf("fail to save node account: %w", err)
				}
			}

		}
		return &cosmos.Result{}, nil
	}

	if (voter.BlockHeight + observeFlex) >= common.BlockHeight(ctx) {
		h.mgr.Slasher().DecSlashPoints(ctx, observeSlashPoints, msg.Signer)
	}

	return &cosmos.Result{}, nil
}
