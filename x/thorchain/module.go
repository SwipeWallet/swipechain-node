package thorchain

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/cosmos/cosmos-sdk/client/context"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	sdkRest "github.com/cosmos/cosmos-sdk/x/auth/client/rest"
	"github.com/cosmos/cosmos-sdk/x/bank"
	"github.com/cosmos/cosmos-sdk/x/supply"
	"github.com/gorilla/mux"
	"github.com/spf13/cobra"
	abci "github.com/tendermint/tendermint/abci/types"

	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/constants"

	"gitlab.com/thorchain/thornode/x/thorchain/client/cli"
	"gitlab.com/thorchain/thornode/x/thorchain/client/rest"
	keeper "gitlab.com/thorchain/thornode/x/thorchain/keeper"
)

// type check to ensure the interface is properly implemented
var (
	_ module.AppModule      = AppModule{}
	_ module.AppModuleBasic = AppModuleBasic{}
)

// app module Basics object
type AppModuleBasic struct{}

func (AppModuleBasic) Name() string {
	return ModuleName
}

func (AppModuleBasic) RegisterCodec(cdc *codec.Codec) {
	RegisterCodec(cdc)
}

func (AppModuleBasic) DefaultGenesis() json.RawMessage {
	return ModuleCdc.MustMarshalJSON(DefaultGenesisState())
}

// Validation check of the Genesis
func (AppModuleBasic) ValidateGenesis(bz json.RawMessage) error {
	var data GenesisState
	err := ModuleCdc.UnmarshalJSON(bz, &data)
	if err != nil {
		return err
	}
	// Once json successfully marshalled, passes along to genesis.go
	return ValidateGenesis(data)
}

// Register rest routes
func (AppModuleBasic) RegisterRESTRoutes(ctx context.CLIContext, rtr *mux.Router) {
	rest.RegisterRoutes(ctx, rtr, StoreKey)
	sdkRest.RegisterTxRoutes(ctx, rtr)
	sdkRest.RegisterRoutes(ctx, rtr, StoreKey)
}

// Get the root query command of this module
func (AppModuleBasic) GetQueryCmd(cdc *codec.Codec) *cobra.Command {
	return cli.GetQueryCmd(StoreKey, cdc)
}

// Get the root tx command of this module
func (AppModuleBasic) GetTxCmd(cdc *codec.Codec) *cobra.Command {
	return cli.GetTxCmd(StoreKey, cdc)
}

type AppModule struct {
	AppModuleBasic
	keeper       keeper.Keeper
	coinKeeper   bank.Keeper
	supplyKeeper supply.Keeper
	mgr          *Mgrs
	keybaseStore KeybaseStore
}

// NewAppModule creates a new AppModule Object
func NewAppModule(k keeper.Keeper, bankKeeper bank.Keeper, supplyKeeper supply.Keeper) AppModule {
	kb, err := getKeybase(os.Getenv("CHAIN_HOME_FOLDER"))
	if err != nil {
		panic(err)
	}
	return AppModule{
		AppModuleBasic: AppModuleBasic{},
		keeper:         k,
		coinKeeper:     bankKeeper,
		supplyKeeper:   supplyKeeper,
		mgr:            NewManagers(k),
		keybaseStore:   kb,
	}
}

func (AppModule) Name() string {
	return ModuleName
}

func (am AppModule) RegisterInvariants(ir sdk.InvariantRegistry) {}

func (am AppModule) Route() string {
	return RouterKey
}

func (am AppModule) NewHandler() sdk.Handler {
	return NewExternalHandler(am.keeper, am.mgr)
}

func (am AppModule) QuerierRoute() string {
	return ModuleName
}

func (am AppModule) NewQuerierHandler() sdk.Querier {
	return NewQuerier(am.keeper, am.keybaseStore)
}

func (am AppModule) BeginBlock(ctx sdk.Context, req abci.RequestBeginBlock) {
	ctx.Logger().Debug("Begin Block", "height", req.Header.Height)
	version := am.keeper.GetLowestActiveVersion(ctx)

	// Does a kvstore migration
	smgr := NewStoreMgr(am.keeper)
	if err := smgr.Iterator(ctx); err != nil {
		os.Exit(10) // halt the chain if unsuccessful
	}

	am.keeper.ClearObservingAddresses(ctx)
	if err := am.mgr.BeginBlock(ctx); err != nil {
		ctx.Logger().Error("fail to get managers", "error", err)
	}
	am.mgr.GasMgr().BeginBlock()

	constantValues := constants.GetConstantValues(version)
	if constantValues == nil {
		ctx.Logger().Error(fmt.Sprintf("constants for version(%s) is not available", version))
		return
	}

	am.mgr.Slasher().BeginBlock(ctx, req, constantValues)

	if err := am.mgr.ValidatorMgr().BeginBlock(ctx, constantValues); err != nil {
		ctx.Logger().Error("Fail to begin block on validator", "error", err)
	}
}

func (am AppModule) EndBlock(ctx sdk.Context, req abci.RequestEndBlock) []abci.ValidatorUpdate {
	ctx.Logger().Debug("End Block", "height", req.Height)

	version := am.keeper.GetLowestActiveVersion(ctx)
	constantValues := constants.GetConstantValues(version)
	if constantValues == nil {
		ctx.Logger().Error(fmt.Sprintf("constants for version(%s) is not available", version))
		return nil
	}
	if err := am.mgr.SwapQ().EndBlock(ctx, am.mgr, version, constantValues); err != nil {
		ctx.Logger().Error("fail to process swap queue", "error", err)
	}

	// slash node accounts for not observing any accepted inbound tx
	if err := am.mgr.Slasher().LackObserving(ctx, constantValues); err != nil {
		ctx.Logger().Error("Unable to slash for lack of observing:", "error", err)
	}
	if err := am.mgr.Slasher().LackSigning(ctx, constantValues, am.mgr); err != nil {
		ctx.Logger().Error("Unable to slash for lack of signing:", "error", err)
	}

	newPoolCycle, err := am.keeper.GetMimir(ctx, constants.NewPoolCycle.String())
	if newPoolCycle < 0 || err != nil {
		newPoolCycle = constantValues.GetInt64Value(constants.NewPoolCycle)
	}
	// Enable a pool every newPoolCycle
	if common.BlockHeight(ctx)%newPoolCycle == 0 && !am.keeper.RagnarokInProgress(ctx) {
		if err := enableNextPool(ctx, am.keeper, am.mgr.EventMgr()); err != nil {
			ctx.Logger().Error("Unable to enable a pool", "error", err)
		}
	}

	am.mgr.ObMgr().EndBlock(ctx, am.keeper)

	// update vault data to account for block rewards and reward units
	if err := am.mgr.VaultMgr().UpdateVaultData(ctx, constantValues, am.mgr.GasMgr(), am.mgr.EventMgr()); err != nil {
		ctx.Logger().Error("fail to update vault data", "error", err)
	}

	if err := am.mgr.VaultMgr().EndBlock(ctx, am.mgr, constantValues); err != nil {
		ctx.Logger().Error("fail to end block for vault manager", "error", err)
	}

	validators := am.mgr.ValidatorMgr().EndBlock(ctx, am.mgr, constantValues)

	// Fill up Yggdrasil vaults
	// We do this AFTER validatorMgr.EndBlock, because we don't want to send
	// funds to a yggdrasil vault that is being churned out this block.
	if err := am.mgr.YggManager().Fund(ctx, am.mgr, constantValues); err != nil {
		ctx.Logger().Error("unable to fund yggdrasil", "error", err)
	}

	am.mgr.GasMgr().EndBlock(ctx, am.keeper, am.mgr.EventMgr())

	return validators
}

func (am AppModule) InitGenesis(ctx sdk.Context, data json.RawMessage) []abci.ValidatorUpdate {
	var genesisState GenesisState
	ModuleCdc.MustUnmarshalJSON(data, &genesisState)
	return InitGenesis(ctx, am.keeper, genesisState)
}

func (am AppModule) ExportGenesis(ctx sdk.Context) json.RawMessage {
	gs := ExportGenesis(ctx, am.keeper)
	return ModuleCdc.MustMarshalJSON(gs)
}
