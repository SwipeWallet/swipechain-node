package thorchain

import (
	"fmt"

	"github.com/blang/semver"

	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/common/cosmos"
	"gitlab.com/thorchain/thornode/constants"
	"gitlab.com/thorchain/thornode/x/thorchain/keeper"
)

// TssKeysignHandler is design to process MsgTssKeysignFail
type TssKeysignHandler struct {
	keeper keeper.Keeper
	mgr    Manager
}

// NewTssKeysignHandler create a new instance of TssKeysignHandler
// when a signer fail to join tss keysign , thorchain need to slash the node account
func NewTssKeysignHandler(keeper keeper.Keeper, mgr Manager) TssKeysignHandler {
	return TssKeysignHandler{
		keeper: keeper,
		mgr:    mgr,
	}
}

// Run is the main entry to process MsgTssKeysignFail
func (h TssKeysignHandler) Run(ctx cosmos.Context, m cosmos.Msg, version semver.Version, constAccessor constants.ConstantValues) (*cosmos.Result, error) {
	msg, ok := m.(MsgTssKeysignFail)
	if !ok {
		return nil, errInvalidMessage
	}
	err := h.validate(ctx, msg, version)
	if err != nil {
		ctx.Logger().Error("MsgTssKeysignFail failed validation", "error", err)
		return nil, err
	}
	result, err := h.handle(ctx, msg, version, constAccessor)
	if err != nil {
		ctx.Logger().Error("failed to process MsgTssKeysignFail", "error", err)
	}
	return result, err
}

func (h TssKeysignHandler) validate(ctx cosmos.Context, msg MsgTssKeysignFail, version semver.Version) error {
	if version.GTE(semver.MustParse("0.18.0")) {
		return h.validateV18(ctx, msg)
	} else if version.GTE(semver.MustParse("0.1.0")) {
		return h.validateV1(ctx, msg)
	}
	return errBadVersion
}

func (h TssKeysignHandler) validateV1(ctx cosmos.Context, msg MsgTssKeysignFail) error {
	if err := msg.ValidateBasic(); err != nil {
		return err
	}

	if !isSignedByActiveNodeAccounts(ctx, h.keeper, msg.GetSigners()) {
		return cosmos.ErrUnauthorized("not authorized")
	}

	active, err := h.keeper.ListActiveNodeAccounts(ctx)
	if err != nil {
		return wrapError(ctx, err, "fail to get list of active node accounts")
	}

	if !HasSimpleMajority(len(active)-len(msg.Blame.BlameNodes), len(active)) {
		ctx.Logger().Error("blame cast too wide", "blame", len(msg.Blame.BlameNodes))
		return fmt.Errorf("blame cast too wide: %d/%d", len(msg.Blame.BlameNodes), len(active))
	}

	return nil
}

func (h TssKeysignHandler) validateV18(ctx cosmos.Context, msg MsgTssKeysignFail) error {
	if err := msg.ValidateBasic(); err != nil {
		return err
	}

	if !isSignedByActiveNodeAccounts(ctx, h.keeper, msg.GetSigners()) {
		shouldAccept := false
		vaults, err := h.keeper.GetAsgardVaultsByStatus(ctx, RetiringVault)
		if err != nil {
			return ErrInternal(err, "fail to get retiring vaults")
		}
		if len(vaults) > 0 {
			for _, signer := range msg.GetSigners() {
				nodeAccount, err := h.keeper.GetNodeAccount(ctx, signer)
				if err != nil {
					return ErrInternal(err, "fail to get node account")
				}

				for _, v := range vaults {
					if v.Membership.Contains(nodeAccount.PubKeySet.Secp256k1) {
						shouldAccept = true
						break
					}
				}
				if shouldAccept {
					break
				}
			}
		}
		if !shouldAccept {
			return cosmos.ErrUnauthorized("not authorized")
		}
		ctx.Logger().Info("keysign failure message from retiring vault member, should accept")
	}

	active, err := h.keeper.ListActiveNodeAccounts(ctx)
	if err != nil {
		return wrapError(ctx, err, "fail to get list of active node accounts")
	}

	if !HasSimpleMajority(len(active)-len(msg.Blame.BlameNodes), len(active)) {
		ctx.Logger().Error("blame cast too wide", "blame", len(msg.Blame.BlameNodes))
		return fmt.Errorf("blame cast too wide: %d/%d", len(msg.Blame.BlameNodes), len(active))
	}

	return nil
}

func (h TssKeysignHandler) handle(ctx cosmos.Context, msg MsgTssKeysignFail, version semver.Version, constAccessor constants.ConstantValues) (*cosmos.Result, error) {
	ctx.Logger().Info("handle MsgTssKeysignFail request", "ID", msg.ID, "signer", msg.Signer, "pubkey", msg.PubKey, "blame", msg.Blame.String())
	if version.GTE(semver.MustParse("0.18.0")) {
		return h.handleV18(ctx, msg, version, constAccessor)
	} else if version.GTE(semver.MustParse("0.13.0")) {
		return h.handleV13(ctx, msg, version, constAccessor)
	} else if version.GTE(semver.MustParse("0.1.0")) {
		return h.handleV1(ctx, msg, version, constAccessor)
	}
	return nil, errBadVersion
}

func (h TssKeysignHandler) handleV1(ctx cosmos.Context, msg MsgTssKeysignFail, version semver.Version, constAccessor constants.ConstantValues) (*cosmos.Result, error) {
	active, err := h.keeper.ListActiveNodeAccounts(ctx)
	if err != nil {
		return nil, wrapError(ctx, err, "fail to get list of active node accounts")
	}

	voter, err := h.keeper.GetTssKeysignFailVoter(ctx, msg.ID)
	if err != nil {
		return nil, err
	}
	observeSlashPoints := constAccessor.GetInt64Value(constants.ObserveSlashPoints)
	h.mgr.Slasher().IncSlashPoints(ctx, observeSlashPoints, msg.Signer)
	if !voter.Sign(msg.Signer) {
		ctx.Logger().Info("signer already signed MsgTssKeysignFail", "signer", msg.Signer.String(), "txid", msg.ID)
		return &cosmos.Result{}, nil
	}
	h.keeper.SetTssKeysignFailVoter(ctx, voter)
	// doesn't have consensus yet
	if !voter.HasConsensus(active) {
		ctx.Logger().Info("not having consensus yet, return")
		return &cosmos.Result{}, nil
	}
	ctx.Logger().Info("has tss keysign consensus!!")

	h.mgr.Slasher().DecSlashPoints(ctx, observeSlashPoints, voter.Signers...)
	voter.Signers = nil
	h.keeper.SetTssKeysignFailVoter(ctx, voter)

	slashPoints := constAccessor.GetInt64Value(constants.FailKeySignSlashPoints)
	// fail to generate a new tss key let's slash the node account

	for _, node := range msg.Blame.BlameNodes {
		nodePubKey, err := common.NewPubKey(node.Pubkey)
		if err != nil {
			return nil, ErrInternal(err, "fail to parse pubkey")
		}
		na, err := h.keeper.GetNodeAccountByPubKey(ctx, nodePubKey)
		if err != nil {
			return nil, ErrInternal(err, fmt.Sprintf("fail to get node account,pub key: %s", nodePubKey.String()))
		}
		if err := h.keeper.IncNodeAccountSlashPoints(ctx, na.NodeAddress, slashPoints); err != nil {
			ctx.Logger().Error("fail to inc slash points", "error", err)
		}

		// go to jail
		ctx.Logger().Info("jailing node", "pubkey", na.PubKeySet.Secp256k1)
		jailTime := constAccessor.GetInt64Value(constants.JailTimeKeysign)
		releaseHeight := common.BlockHeight(ctx) + jailTime
		reason := "failed to perform keysign"
		if err := h.keeper.SetNodeAccountJail(ctx, na.NodeAddress, releaseHeight, reason); err != nil {
			ctx.Logger().Error("fail to set node account jail", "node address", na.NodeAddress, "reason", reason, "error", err)
		}
	}
	if err := h.updateVaultKeySign(ctx, msg.PubKey); err != nil {
		ctx.Logger().Error("fail to update vault signing party", "error", err)
	}
	return &cosmos.Result{}, nil
}

// handleV13 is introduced at 0.13.0 version, which change the way how SimplyMajority get calculated
func (h TssKeysignHandler) handleV13(ctx cosmos.Context, msg MsgTssKeysignFail, version semver.Version, constAccessor constants.ConstantValues) (*cosmos.Result, error) {
	active, err := h.keeper.ListActiveNodeAccounts(ctx)
	if err != nil {
		return nil, wrapError(ctx, err, "fail to get list of active node accounts")
	}

	voter, err := h.keeper.GetTssKeysignFailVoter(ctx, msg.ID)
	if err != nil {
		return nil, err
	}
	observeSlashPoints := constAccessor.GetInt64Value(constants.ObserveSlashPoints)
	h.mgr.Slasher().IncSlashPoints(ctx, observeSlashPoints, msg.Signer)
	if !voter.Sign(msg.Signer) {
		ctx.Logger().Info("signer already signed MsgTssKeysignFail", "signer", msg.Signer.String(), "txid", msg.ID)
		return &cosmos.Result{}, nil
	}
	h.keeper.SetTssKeysignFailVoter(ctx, voter)
	// doesn't have consensus yet
	if !voter.HasConsensusV13(active) {
		ctx.Logger().Info("not having consensus yet, return")
		return &cosmos.Result{}, nil
	}
	ctx.Logger().Info("has tss keysign consensus!!")

	h.mgr.Slasher().DecSlashPoints(ctx, observeSlashPoints, voter.Signers...)
	voter.Signers = nil
	h.keeper.SetTssKeysignFailVoter(ctx, voter)

	slashPoints := constAccessor.GetInt64Value(constants.FailKeySignSlashPoints)
	// fail to generate a new tss key let's slash the node account

	for _, node := range msg.Blame.BlameNodes {
		nodePubKey, err := common.NewPubKey(node.Pubkey)
		if err != nil {
			return nil, ErrInternal(err, "fail to parse pubkey")
		}
		na, err := h.keeper.GetNodeAccountByPubKey(ctx, nodePubKey)
		if err != nil {
			return nil, ErrInternal(err, fmt.Sprintf("fail to get node account,pub key: %s", nodePubKey.String()))
		}
		if err := h.keeper.IncNodeAccountSlashPoints(ctx, na.NodeAddress, slashPoints); err != nil {
			ctx.Logger().Error("fail to inc slash points", "error", err)
		}

		// go to jail
		ctx.Logger().Info("jailing node", "pubkey", na.PubKeySet.Secp256k1)
		jailTime := constAccessor.GetInt64Value(constants.JailTimeKeysign)
		releaseHeight := common.BlockHeight(ctx) + jailTime
		reason := "failed to perform keysign"
		if err := h.keeper.SetNodeAccountJail(ctx, na.NodeAddress, releaseHeight, reason); err != nil {
			ctx.Logger().Error("fail to set node account jail", "node address", na.NodeAddress, "reason", reason, "error", err)
		}
	}
	if err := h.updateVaultKeySign(ctx, msg.PubKey); err != nil {
		ctx.Logger().Error("fail to update vault signing party", "error", err)
	}
	return &cosmos.Result{}, nil
}

// handleV18 is introduced at 0.18.0 version, which change the way how SimplyMajority get calculated
func (h TssKeysignHandler) handleV18(ctx cosmos.Context, msg MsgTssKeysignFail, version semver.Version, constAccessor constants.ConstantValues) (*cosmos.Result, error) {
	voter, err := h.keeper.GetTssKeysignFailVoter(ctx, msg.ID)
	if err != nil {
		return nil, err
	}
	observeSlashPoints := constAccessor.GetInt64Value(constants.ObserveSlashPoints)
	h.mgr.Slasher().IncSlashPoints(ctx, observeSlashPoints, msg.Signer)
	if !voter.Sign(msg.Signer) {
		ctx.Logger().Info("signer already signed MsgTssKeysignFail", "signer", msg.Signer.String(), "txid", msg.ID)
		return &cosmos.Result{}, nil
	}
	h.keeper.SetTssKeysignFailVoter(ctx, voter)

	vault, err := h.keeper.GetVault(ctx, msg.PubKey)
	if err != nil {
		return nil, wrapError(ctx, err, "fail to get vault")
	}
	if vault.IsEmpty() {
		return &cosmos.Result{}, nil
	}
	var vaultMemberNodes NodeAccounts
	for _, item := range vault.Membership {
		addr, err := item.GetThorAddress()
		if err != nil {
			return nil, wrapError(ctx, err, "fail to get thor address for "+item.String())
		}
		na, err := h.keeper.GetNodeAccount(ctx, addr)
		if err != nil {
			return nil, wrapError(ctx, err, "fail to get node account")
		}
		vaultMemberNodes = append(vaultMemberNodes, na)
	}
	// doesn't have consensus yet
	if !voter.HasConsensusV18(vaultMemberNodes) {
		ctx.Logger().Info("not having consensus yet, return")
		return &cosmos.Result{}, nil
	}
	ctx.Logger().Info("has tss keysign consensus!!")

	h.mgr.Slasher().DecSlashPoints(ctx, observeSlashPoints, voter.Signers...)
	voter.Signers = nil
	h.keeper.SetTssKeysignFailVoter(ctx, voter)

	slashPoints := constAccessor.GetInt64Value(constants.FailKeySignSlashPoints)
	// fail to generate a new tss key let's slash the node account

	for _, node := range msg.Blame.BlameNodes {
		nodePubKey, err := common.NewPubKey(node.Pubkey)
		if err != nil {
			return nil, ErrInternal(err, "fail to parse pubkey")
		}
		na, err := h.keeper.GetNodeAccountByPubKey(ctx, nodePubKey)
		if err != nil {
			return nil, ErrInternal(err, fmt.Sprintf("fail to get node account,pub key: %s", nodePubKey.String()))
		}
		if err := h.keeper.IncNodeAccountSlashPoints(ctx, na.NodeAddress, slashPoints); err != nil {
			ctx.Logger().Error("fail to inc slash points", "error", err)
		}

		// go to jail
		ctx.Logger().Info("jailing node", "pubkey", na.PubKeySet.Secp256k1)
		jailTime := constAccessor.GetInt64Value(constants.JailTimeKeysign)
		releaseHeight := common.BlockHeight(ctx) + jailTime
		reason := "failed to perform keysign"
		if err := h.keeper.SetNodeAccountJail(ctx, na.NodeAddress, releaseHeight, reason); err != nil {
			ctx.Logger().Error("fail to set node account jail", "node address", na.NodeAddress, "reason", reason, "error", err)
		}
	}
	if err := h.updateVaultKeySign(ctx, msg.PubKey); err != nil {
		ctx.Logger().Error("fail to update vault signing party", "error", err)
	}
	return &cosmos.Result{}, nil
}

func (h TssKeysignHandler) updateVaultKeySign(ctx cosmos.Context, vaultPubKey common.PubKey) error {
	accountAddrs, err := h.keeper.GetObservingAddresses(ctx)
	if err != nil {
		return fmt.Errorf("fail to get observing addresses: %w", err)
	}

	vault, err := h.keeper.GetVault(ctx, vaultPubKey)
	if err != nil {
		return fmt.Errorf("fail to get vault: %w", err)
	}
	members := vault.Membership
	threshold, err := GetThreshold(len(vault.Membership))
	if err != nil {
		return fmt.Errorf("fail to get threshold: %w", err)
	}
	totalObservingAccounts := len(accountAddrs)
	if totalObservingAccounts > 0 && totalObservingAccounts >= threshold {
		members, err = vault.GetMembers(accountAddrs)
		if err != nil {
			return fmt.Errorf("fail to get signers: %w", err)
		}
	}

	// build signer list, exclude any node accounts in jail
	signers := make(common.PubKeys, 0)
	signers = h.getSignerCandidates(ctx, signers, members, NodeActive)

	// not enough nodes to form keysign party , try to find some nodes that is not actively observing , and not jailed
	// when scale down , the actively observing validator set will become smaller
	if len(signers) < threshold {
		signers = h.getSignerCandidates(ctx, signers, vault.Membership, NodeReady)
		signers = h.getSignerCandidates(ctx, signers, vault.Membership, NodeStandby)
	}

	// still doesn't have enough signer, let's add those node in Disable
	if len(signers) < threshold {
		signers = h.getSignerCandidates(ctx, signers, vault.Membership, NodeDisabled)
	}
	// still don't have enough signer , jail free
	if len(signers) < threshold {
		signers = vault.Membership
	}
	// if there are 9 nodes in total , it need 6 nodes to sign a message
	// 3 signer send request to thorchain at block height 100
	// another 3 signer send request to thorchain at block height 101
	// in this case we get into trouble ,they get different results, key sign is going to fail
	signerParty, err := ChooseSignerParty(signers, common.BlockHeight(ctx), len(vault.Membership))
	if err != nil {
		return fmt.Errorf("fail to choose signer party members: %w", err)
	}
	vault.SigningParty = signerParty
	return h.keeper.SetVault(ctx, vault)
}

func (h TssKeysignHandler) getSignerCandidates(ctx cosmos.Context, signers, candidates common.PubKeys, status NodeStatus) common.PubKeys {
	for _, mem := range candidates {

		if signers.Contains(mem) {
			continue
		}
		na, err := h.keeper.GetNodeAccountByPubKey(ctx, mem)
		if err != nil {
			ctx.Logger().Error("fail to get node account", "error", err)
			continue
		}

		if na.Status != status {
			continue
		}

		jail, err := h.keeper.GetNodeAccountJail(ctx, na.NodeAddress)
		if err != nil {
			ctx.Logger().Error("fail to get node account jail", "error", err)
		}
		if jail.IsJailed(ctx) {
			continue
		}
		signers = append(signers, mem)
	}
	return signers
}
