package thorchain

import (
	"fmt"

	"github.com/blang/semver"

	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/common/cosmos"
	"gitlab.com/thorchain/thornode/constants"
	"gitlab.com/thorchain/thornode/x/thorchain/keeper"
)

// SetNodeKeysHandler process MsgSetNodeKeys
// MsgSetNodeKeys is used by operators after the node account had been white list , to update the consensus pubkey and node account pubkey
type SetNodeKeysHandler struct {
	keeper keeper.Keeper
	mgr    Manager
}

// NewSetNodeKeysHandler create a new instance of SetNodeKeysHandler
func NewSetNodeKeysHandler(keeper keeper.Keeper, mgr Manager) SetNodeKeysHandler {
	return SetNodeKeysHandler{
		keeper: keeper,
		mgr:    mgr,
	}
}

// Run is the main entry point to process MsgSetNodeKeys
func (h SetNodeKeysHandler) Run(ctx cosmos.Context, m cosmos.Msg, version semver.Version, constAccessor constants.ConstantValues) (*cosmos.Result, error) {
	msg, ok := m.(MsgSetNodeKeys)
	if !ok {
		return nil, errInvalidMessage
	}
	if err := h.validate(ctx, msg, version, constAccessor); err != nil {
		ctx.Logger().Error("MsgSetNodeKeys failed validation", "error", err)
		return nil, err
	}
	result, err := h.handle(ctx, msg, version, constAccessor)
	if err != nil {
		ctx.Logger().Error("fail to process MsgSetNodeKey", "error", err)
	}
	return result, err
}

func (h SetNodeKeysHandler) validate(ctx cosmos.Context, msg MsgSetNodeKeys, version semver.Version, constAccessor constants.ConstantValues) error {
	if version.GTE(semver.MustParse("0.1.0")) {
		return h.validateV1(ctx, msg, constAccessor)
	}
	return errInvalidVersion
}

func (h SetNodeKeysHandler) validateV1(ctx cosmos.Context, msg MsgSetNodeKeys, constAccessor constants.ConstantValues) error {
	if err := msg.ValidateBasic(); err != nil {
		return err
	}

	nodeAccount, err := h.keeper.GetNodeAccount(ctx, msg.Signer)
	if err != nil {
		return cosmos.ErrUnauthorized(fmt.Sprintf("fail to get node account(%s):%s", msg.Signer.String(), err)) // notAuthorized
	}
	if nodeAccount.IsEmpty() {
		return cosmos.ErrUnauthorized(fmt.Sprintf("unauthorized account(%s)", msg.Signer))
	}

	cost := constAccessor.GetInt64Value(constants.CliTxCost)
	if nodeAccount.Bond.LT(cosmos.NewUint(uint64(cost))) {
		return cosmos.ErrUnauthorized("not enough bond")
	}

	// You should not able to update node address when the node is in active mode
	// for example if they update observer address
	if nodeAccount.Status == NodeActive {
		return fmt.Errorf("node %s is active, so it can't update itself", nodeAccount.NodeAddress)
	}
	if nodeAccount.Status == NodeDisabled {
		return fmt.Errorf("node %s is disabled, so it can't update itself", nodeAccount.NodeAddress)
	}
	if err := h.keeper.EnsureNodeKeysUnique(ctx, msg.ValidatorConsPubKey, msg.PubKeySetSet); err != nil {
		return err
	}

	return nil
}

func (h SetNodeKeysHandler) handle(ctx cosmos.Context, msg MsgSetNodeKeys, version semver.Version, constAccessor constants.ConstantValues) (*cosmos.Result, error) {
	ctx.Logger().Info("handleMsgSetNodeKeys request")
	if version.GTE(semver.MustParse("0.1.0")) {
		return h.handleV1(ctx, msg, version, constAccessor)
	}
	return nil, errBadVersion
}

// handleV1 a message to set node keys
func (h SetNodeKeysHandler) handleV1(ctx cosmos.Context, msg MsgSetNodeKeys, version semver.Version, constAccessor constants.ConstantValues) (*cosmos.Result, error) {
	nodeAccount, err := h.keeper.GetNodeAccount(ctx, msg.Signer)
	if err != nil {
		ctx.Logger().Error("fail to get node account", "error", err, "address", msg.Signer.String())
		return nil, cosmos.ErrUnauthorized(fmt.Sprintf("%s is not authorized", msg.Signer))
	}

	cost := cosmos.NewUint(uint64(constAccessor.GetInt64Value(constants.CliTxCost)))
	if cost.GT(nodeAccount.Bond) {
		cost = nodeAccount.Bond
	}
	// Here make sure THORNode don't change the node account's bond
	nodeAccount.UpdateStatus(NodeStandby, common.BlockHeight(ctx))
	nodeAccount.PubKeySet = msg.PubKeySetSet
	nodeAccount.Bond = common.SafeSub(nodeAccount.Bond, cost)
	nodeAccount.ValidatorConsPubKey = msg.ValidatorConsPubKey
	if err := h.keeper.SetNodeAccount(ctx, nodeAccount); err != nil {
		return nil, fmt.Errorf("fail to save node account: %w", err)
	}

	// add 10 bond to reserve
	if common.RuneAsset().Chain.Equals(common.THORChain) {
		coin := common.NewCoin(common.RuneNative, cost)
		if err := h.keeper.SendFromAccountToModule(ctx, msg.Signer, ReserveName, coin); err != nil {
			ctx.Logger().Error("fail to transfer funds from bond to reserve", "error", err)
			return nil, err
		}
	} else {
		vaultData, err := h.keeper.GetVaultData(ctx)
		if err != nil {
			return nil, fmt.Errorf("fail to get vault data: %w", err)
		}
		vaultData.TotalReserve = vaultData.TotalReserve.Add(cost)
		if err := h.keeper.SetVaultData(ctx, vaultData); err != nil {
			return nil, fmt.Errorf("fail to save vault data: %w", err)
		}
	}

	ctx.EventManager().EmitEvent(
		cosmos.NewEvent("set_node_keys",
			cosmos.NewAttribute("node_address", msg.Signer.String()),
			cosmos.NewAttribute("node_secp256k1_pubkey", msg.PubKeySetSet.Secp256k1.String()),
			cosmos.NewAttribute("node_ed25519_pubkey", msg.PubKeySetSet.Ed25519.String()),
			cosmos.NewAttribute("validator_consensus_pub_key", msg.ValidatorConsPubKey)))

	return &cosmos.Result{}, nil
}
