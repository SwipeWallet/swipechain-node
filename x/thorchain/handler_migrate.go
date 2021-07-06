package thorchain

import (
	"github.com/blang/semver"

	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/common/cosmos"
	"gitlab.com/thorchain/thornode/constants"
	"gitlab.com/thorchain/thornode/x/thorchain/keeper"
)

// MigrateHandler is a handler to process MsgMigrate
type MigrateHandler struct {
	keeper keeper.Keeper
	mgr    Manager
}

// NewMigrateHandler create a new instance of MigrateHandler
func NewMigrateHandler(keeper keeper.Keeper, mgr Manager) MigrateHandler {
	return MigrateHandler{
		keeper: keeper,
		mgr:    mgr,
	}
}

// Run is the main entry point of Migrate handler
func (h MigrateHandler) Run(ctx cosmos.Context, m cosmos.Msg, version semver.Version, _ constants.ConstantValues) (*cosmos.Result, error) {
	msg, ok := m.(MsgMigrate)
	if !ok {
		return nil, errInvalidMessage
	}
	if err := h.validate(ctx, msg, version); err != nil {
		return nil, err
	}
	return h.handle(ctx, msg, version)
}

func (h MigrateHandler) validate(ctx cosmos.Context, msg MsgMigrate, version semver.Version) error {
	if version.GTE(semver.MustParse("0.1.0")) {
		return h.validateV1(ctx, msg)
	}
	return errInvalidVersion
}

func (h MigrateHandler) validateV1(ctx cosmos.Context, msg MsgMigrate) error {
	if err := msg.ValidateBasic(); nil != err {
		return err
	}
	return nil
}

func (h MigrateHandler) handle(ctx cosmos.Context, msg MsgMigrate, version semver.Version) (*cosmos.Result, error) {
	ctx.Logger().Info("receive MsgMigrate", "request tx hash", msg.Tx.Tx.ID)
	if version.GTE(semver.MustParse("0.1.0")) {
		return h.handleV1(ctx, version, msg)
	}
	return nil, errBadVersion
}

func (h MigrateHandler) slash(ctx cosmos.Context, version semver.Version, tx ObservedTx) error {
	var returnErr error
	for _, c := range tx.Tx.Coins {
		if err := h.mgr.Slasher().SlashNodeAccount(ctx, tx.ObservedPubKey, c.Asset, c.Amount, h.mgr); err != nil {
			ctx.Logger().Error("fail to slash account", "error", err)
			returnErr = err
		}
	}
	return returnErr
}

func (h MigrateHandler) handleV1(ctx cosmos.Context, version semver.Version, msg MsgMigrate) (*cosmos.Result, error) {
	// update txOut record with our TxID that sent funds out of the pool
	txOut, err := h.keeper.GetTxOut(ctx, msg.BlockHeight)
	if err != nil {
		ctx.Logger().Error("unable to get txOut record", "error", err)
		return nil, cosmos.ErrUnknownRequest(err.Error())
	}

	shouldSlash := true
	for i, tx := range txOut.TxArray {
		// migrate is the memo used by thorchain to identify fund migration between asgard vault.
		// it use migrate:{block height} to mark a tx out caused by vault rotation
		// this type of tx out is special , because it doesn't have relevant tx in to trigger it, it is trigger by thorchain itself.
		fromAddress, _ := tx.VaultPubKey.GetAddress(tx.Chain)
		if tx.InHash.Equals(common.BlankTxID) &&
			tx.OutHash.IsEmpty() &&
			msg.Tx.Tx.Coins.Contains(tx.Coin) &&
			tx.ToAddress.Equals(msg.Tx.Tx.ToAddress) &&
			fromAddress.Equals(msg.Tx.Tx.FromAddress) {

			txOut.TxArray[i].OutHash = msg.Tx.Tx.ID
			shouldSlash = false

			if err := h.keeper.SetTxOut(ctx, txOut); nil != err {
				return nil, ErrInternal(err, "fail to save tx out")
			}

			break
		}
	}

	if shouldSlash {
		ctx.Logger().Info("slash node account,migration has no matched txout", "outbound tx", msg.Tx.Tx)
		if err := h.slash(ctx, version, msg.Tx); err != nil {
			return nil, ErrInternal(err, "fail to slash account")
		}
	}

	if err := h.keeper.SetLastSignedHeight(ctx, msg.BlockHeight); err != nil {
		ctx.Logger().Error("fail to update last signed height", "error", err)
	}

	return &cosmos.Result{}, nil
}
