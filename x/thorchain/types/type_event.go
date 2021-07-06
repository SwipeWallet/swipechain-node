package types

import (
	"fmt"
	"strconv"

	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/common/cosmos"
)

// all event types support by THORChain
const (
	SwapEventType     = `swap`
	StakeEventType    = `stake`
	UnstakeEventType  = `unstake`
	AddEventType      = `add`
	PoolEventType     = `pool`
	RewardEventType   = `rewards`
	RefundEventType   = `refund`
	BondEventType     = `bond`
	GasEventType      = `gas`
	ReserveEventType  = `reserve`
	SlashEventType    = `slash`
	ErrataEventType   = `errata`
	FeeEventType      = `fee`
	OutboundEventType = `outbound`
)

// PoolMod pool modifications
type PoolMod struct {
	Asset    common.Asset `json:"asset"`
	RuneAmt  cosmos.Uint  `json:"rune_amt"`
	RuneAdd  bool         `json:"rune_add"`
	AssetAmt cosmos.Uint  `json:"asset_amt"`
	AssetAdd bool         `json:"asset_add"`
}

// PoolMods a list of pool modifications
type PoolMods []PoolMod

// NewPoolMod create a new instance of PoolMod
func NewPoolMod(asset common.Asset, runeAmt cosmos.Uint, runeAdd bool, assetAmt cosmos.Uint, assetAdd bool) PoolMod {
	return PoolMod{
		Asset:    asset,
		RuneAmt:  runeAmt,
		RuneAdd:  runeAdd,
		AssetAmt: assetAmt,
		AssetAdd: assetAdd,
	}
}

// EventSwap event for swap action
type EventSwap struct {
	Pool               common.Asset `json:"pool"`
	PriceTarget        cosmos.Uint  `json:"price_target"`
	TradeSlip          cosmos.Uint  `json:"trade_slip"`
	LiquidityFee       cosmos.Uint  `json:"liquidity_fee"`
	LiquidityFeeInRune cosmos.Uint  `json:"liquidity_fee_in_rune"`
	InTx               common.Tx    `json:"in_tx"` // this is the Tx that cause the swap to happen, if it is a double swap , then the txid will be blank
	OutTxs             common.Tx    `json:"out_txs"`
}

// NewEventSwap create a new swap event
func NewEventSwap(pool common.Asset, priceTarget, fee, tradeSlip, liquidityFeeInRune cosmos.Uint, inTx common.Tx) EventSwap {
	return EventSwap{
		Pool:               pool,
		PriceTarget:        priceTarget,
		TradeSlip:          tradeSlip,
		LiquidityFee:       fee,
		LiquidityFeeInRune: liquidityFeeInRune,
		InTx:               inTx,
	}
}

// Type return a string that represent the type, it should not duplicated with other event
func (e EventSwap) Type() string {
	return SwapEventType
}

// Events convert EventSwap to key value pairs used in cosmos
func (e EventSwap) Events() (cosmos.Events, error) {
	evt := cosmos.NewEvent(e.Type(),
		cosmos.NewAttribute("pool", e.Pool.String()),
		cosmos.NewAttribute("price_target", e.PriceTarget.String()),
		cosmos.NewAttribute("trade_slip", e.TradeSlip.String()),
		cosmos.NewAttribute("liquidity_fee", e.LiquidityFee.String()),
		cosmos.NewAttribute("liquidity_fee_in_rune", e.LiquidityFeeInRune.String()),
	)
	evt = evt.AppendAttributes(e.InTx.ToAttributes()...)
	return cosmos.Events{evt}, nil
}

// EventStake stake event
type EventStake struct {
	Pool        common.Asset   `json:"pool"`
	StakeUnits  cosmos.Uint    `json:"stake_units"`
	RuneAddress common.Address `json:"rune_address"`
	RuneAmount  cosmos.Uint    `json:"rune_amount"`
	AssetAmount cosmos.Uint    `json:"asset_amount"`
	RuneTxID    common.TxID    `json:"rune_tx_id"`
	AssetTxID   common.TxID    `json:"asset_tx_id"`
}

// NewEventStake create a new stake event
func NewEventStake(pool common.Asset,
	su cosmos.Uint,
	runeAddress common.Address,
	runeAmount, assetAmount cosmos.Uint, runeTxID, assetTxID common.TxID) EventStake {
	return EventStake{
		Pool:        pool,
		StakeUnits:  su,
		RuneAddress: runeAddress,
		RuneAmount:  runeAmount,
		AssetAmount: assetAmount,
		RuneTxID:    runeTxID,
		AssetTxID:   assetTxID,
	}
}

// Type return the event type
func (e EventStake) Type() string {
	return StakeEventType
}

// Events return cosmos.Events which is cosmos.Attribute(key value pairs)
func (e EventStake) Events() (cosmos.Events, error) {
	evt := cosmos.NewEvent(e.Type(),
		cosmos.NewAttribute("pool", e.Pool.String()),
		cosmos.NewAttribute("stake_units", e.StakeUnits.String()),
		cosmos.NewAttribute("rune_address", e.RuneAddress.String()),
		cosmos.NewAttribute("rune_amount", e.RuneAmount.String()),
		cosmos.NewAttribute("asset_amount", e.AssetAmount.String()),
	)
	if !e.RuneTxID.Equals(e.AssetTxID) {
		evt = evt.AppendAttributes(cosmos.NewAttribute(fmt.Sprintf("%s_txid", common.RuneAsset().Chain), e.RuneTxID.String()))
	}

	evt = evt.AppendAttributes(cosmos.NewAttribute(fmt.Sprintf("%s_txid", e.Pool.Chain), e.AssetTxID.String()))
	return cosmos.Events{
		evt,
	}, nil
}

// EventUnstake represent unstake
type EventUnstake struct {
	Pool        common.Asset `json:"pool"`
	StakeUnits  cosmos.Uint  `json:"stake_units"`
	BasisPoints int64        `json:"basis_points"` // 1 ==> 10,0000
	Asymmetry   cosmos.Dec   `json:"asymmetry"`    // -1.0 <==> 1.0
	InTx        common.Tx    `json:"in_tx"`
}

// NewEventUnstake create a new unstake event
func NewEventUnstake(pool common.Asset, su cosmos.Uint, basisPts int64, asym cosmos.Dec, inTx common.Tx) EventUnstake {
	return EventUnstake{
		Pool:        pool,
		StakeUnits:  su,
		BasisPoints: basisPts,
		Asymmetry:   asym,
		InTx:        inTx,
	}
}

// Type return the unstake event type
func (e EventUnstake) Type() string {
	return UnstakeEventType
}

// Events return the cosmos event
func (e EventUnstake) Events() (cosmos.Events, error) {
	evt := cosmos.NewEvent(e.Type(),
		cosmos.NewAttribute("pool", e.Pool.String()),
		cosmos.NewAttribute("stake_units", e.StakeUnits.String()),
		cosmos.NewAttribute("basis_points", strconv.FormatInt(e.BasisPoints, 10)),
		cosmos.NewAttribute("asymmetry", e.Asymmetry.String()))
	evt = evt.AppendAttributes(e.InTx.ToAttributes()...)
	return cosmos.Events{evt}, nil
}

// EventAdd represent add operation
type EventAdd struct {
	Pool common.Asset `json:"pool"`
	InTx common.Tx    `json:"in_tx"`
}

// NewEventAdd create a new add event
func NewEventAdd(pool common.Asset, inTx common.Tx) EventAdd {
	return EventAdd{
		Pool: pool,
		InTx: inTx,
	}
}

// Type return add event type
func (e EventAdd) Type() string {
	return AddEventType
}

// Events get all events
func (e EventAdd) Events() (cosmos.Events, error) {
	evt := cosmos.NewEvent(e.Type(),
		cosmos.NewAttribute("pool", e.Pool.String()))
	evt = evt.AppendAttributes(e.InTx.ToAttributes()...)
	return cosmos.Events{evt}, nil
}

// EventPool represent pool change event
type EventPool struct {
	Pool   common.Asset `json:"pool"`
	Status PoolStatus   `json:"status"`
}

// NewEventPool create a new pool change event
func NewEventPool(pool common.Asset, status PoolStatus) EventPool {
	return EventPool{
		Pool:   pool,
		Status: status,
	}
}

// Type return pool event type
func (e EventPool) Type() string {
	return PoolEventType
}

// Events provide an instance of cosmos.Events
func (e EventPool) Events() (cosmos.Events, error) {
	return cosmos.Events{
		cosmos.NewEvent(e.Type(),
			cosmos.NewAttribute("pool", e.Pool.String()),
			cosmos.NewAttribute("pool_status", e.Status.String())),
	}, nil
}

// PoolAmt pool asset amount
type PoolAmt struct {
	Asset  common.Asset `json:"asset"`
	Amount int64        `json:"amount"`
}

// EventRewards reward event
type EventRewards struct {
	BondReward  cosmos.Uint `json:"bond_reward"`
	PoolRewards []PoolAmt   `json:"pool_rewards"`
}

// NewEventRewards create a new reward event
func NewEventRewards(bondReward cosmos.Uint, poolRewards []PoolAmt) EventRewards {
	return EventRewards{
		BondReward:  bondReward,
		PoolRewards: poolRewards,
	}
}

// Type return reward event type
func (e EventRewards) Type() string {
	return RewardEventType
}

func (e EventRewards) Events() (cosmos.Events, error) {
	evt := cosmos.NewEvent(e.Type(),
		cosmos.NewAttribute("bond_reward", e.BondReward.String()),
	)
	for _, item := range e.PoolRewards {
		evt = evt.AppendAttributes(cosmos.NewAttribute(item.Asset.String(), strconv.FormatInt(item.Amount, 10)))
	}
	return cosmos.Events{evt}, nil
}

// EventRefund represent a refund activity , and contains the reason why it get refund
type EventRefund struct {
	Code   uint32     `json:"code"`
	Reason string     `json:"reason"`
	InTx   common.Tx  `json:"in_tx"`
	Fee    common.Fee `json:"fee"`
}

// NewEventRefund create a new EventRefund
func NewEventRefund(code uint32, reason string, inTx common.Tx, fee common.Fee) EventRefund {
	return EventRefund{
		Code:   code,
		Reason: reason,
		InTx:   inTx,
		Fee:    fee,
	}
}

// Type return reward event type
func (e EventRefund) Type() string {
	return RefundEventType
}

// Events return events
func (e EventRefund) Events() (cosmos.Events, error) {
	evt := cosmos.NewEvent(e.Type(),
		cosmos.NewAttribute("code", strconv.FormatUint(uint64(e.Code), 10)),
		cosmos.NewAttribute("reason", e.Reason),
	)
	evt = evt.AppendAttributes(e.InTx.ToAttributes()...)
	return cosmos.Events{evt}, nil
}

type BondType string

const (
	BondPaid     BondType = `bond_paid`
	BondReturned BondType = `bond_returned`
)

// EventBond bond paid or returned event
type EventBond struct {
	Amount   cosmos.Uint `json:"amount"`
	BondType BondType    `json:"bond_type"`
	TxIn     common.Tx   `json:"tx_in"`
}

// NewEventBond create a new Bond Events
func NewEventBond(amount cosmos.Uint, bondType BondType, txIn common.Tx) EventBond {
	return EventBond{
		Amount:   amount,
		BondType: bondType,
		TxIn:     txIn,
	}
}

// Type return bond event Type
func (e EventBond) Type() string {
	return BondEventType
}

// Events return all the event attributes
func (e EventBond) Events() (cosmos.Events, error) {
	evt := cosmos.NewEvent(e.Type(),
		cosmos.NewAttribute("amount", e.Amount.String()),
		cosmos.NewAttribute("bound_type", string(e.BondType)))
	evt = evt.AppendAttributes(e.TxIn.ToAttributes()...)
	return cosmos.Events{evt}, nil
}

type GasType string

type GasPool struct {
	Asset    common.Asset `json:"asset"`
	AssetAmt cosmos.Uint  `json:"asset_amt"`
	RuneAmt  cosmos.Uint  `json:"rune_amt"`
	Count    int64        `json:"transaction_count"`
}

// EventGas represent the events happened in thorchain related to Gas
type EventGas struct {
	Pools []GasPool `json:"pools"`
}

// NewEventGas create a new EventGas instance
func NewEventGas() *EventGas {
	return &EventGas{
		Pools: make([]GasPool, 0),
	}
}

// UpsertGasPool update the Gas Pools hold by EventGas instance
// if the given gasPool already exist, then it merge the gasPool with internal one , otherwise add it to the list
func (e *EventGas) UpsertGasPool(pool GasPool) {
	for i, p := range e.Pools {
		if p.Asset == pool.Asset {
			e.Pools[i].RuneAmt = p.RuneAmt.Add(pool.RuneAmt)
			e.Pools[i].AssetAmt = p.AssetAmt.Add(pool.AssetAmt)
			return
		}
	}
	e.Pools = append(e.Pools, pool)
}

// Type return event type
func (e *EventGas) Type() string {
	return GasEventType
}

func (e *EventGas) Events() (cosmos.Events, error) {
	events := make(cosmos.Events, 0, len(e.Pools))
	for _, item := range e.Pools {
		evt := cosmos.NewEvent(e.Type(),
			cosmos.NewAttribute("asset", item.Asset.String()),
			cosmos.NewAttribute("asset_amt", item.AssetAmt.String()),
			cosmos.NewAttribute("rune_amt", item.RuneAmt.String()),
			cosmos.NewAttribute("transaction_count", strconv.FormatInt(item.Count, 10)))
		events = append(events, evt)
	}
	return events, nil
}

// EventReserve Reserve event type
type EventReserve struct {
	ReserveContributor ReserveContributor `json:"reserve_contributor"`
	InTx               common.Tx          `json:"in_tx"`
}

// NewEventReserve create a new instance of EventReserve
func NewEventReserve(contributor ReserveContributor, inTx common.Tx) EventReserve {
	return EventReserve{
		ReserveContributor: contributor,
		InTx:               inTx,
	}
}

func (e EventReserve) Type() string {
	return ReserveEventType
}

func (e EventReserve) Events() (cosmos.Events, error) {
	evt := cosmos.NewEvent(e.Type(),
		cosmos.NewAttribute("contributor_address", e.ReserveContributor.Address.String()),
		cosmos.NewAttribute("amount", e.ReserveContributor.Amount.String()),
	)
	evt = evt.AppendAttributes(e.InTx.ToAttributes()...)
	return cosmos.Events{
		evt,
	}, nil
}

// EventSlash represent a change in pool balance which caused by slash a node account
type EventSlash struct {
	Pool        common.Asset `json:"pool"`
	SlashAmount []PoolAmt    `json:"slash_amount"`
}

func NewEventSlash(pool common.Asset, slashAmount []PoolAmt) EventSlash {
	return EventSlash{
		Pool:        pool,
		SlashAmount: slashAmount,
	}
}

// Type return slash event type
func (e EventSlash) Type() string {
	return SlashEventType
}

func (e EventSlash) Events() (cosmos.Events, error) {
	evt := cosmos.NewEvent(e.Type(),
		cosmos.NewAttribute("pool", e.Pool.String()))
	for _, item := range e.SlashAmount {
		evt.AppendAttributes(cosmos.NewAttribute(item.Asset.String(), strconv.FormatInt(item.Amount, 10)))
	}
	return cosmos.Events{evt}, nil
}

// EventErrata represent a change in pool balance which caused by an errata transaction
type EventErrata struct {
	TxID  common.TxID `json:"tx_id"`
	Pools PoolMods    `json:"pools"`
}

func NewEventErrata(txID common.TxID, pools PoolMods) EventErrata {
	return EventErrata{
		TxID:  txID,
		Pools: pools,
	}
}

// Type return slash event type
func (e EventErrata) Type() string {
	return ErrataEventType
}

// Events return a cosmos.Events type
func (e EventErrata) Events() (cosmos.Events, error) {
	events := make(cosmos.Events, 0, len(e.Pools))
	for _, item := range e.Pools {
		evt := cosmos.NewEvent(e.Type(),
			cosmos.NewAttribute("in_tx_id", e.TxID.String()),
			cosmos.NewAttribute("asset", item.Asset.String()),
			cosmos.NewAttribute("rune_amt", item.RuneAmt.String()),
			cosmos.NewAttribute("rune_add", strconv.FormatBool(item.RuneAdd)),
			cosmos.NewAttribute("asset_amt", item.AssetAmt.String()),
			cosmos.NewAttribute("asset_add", strconv.FormatBool(item.AssetAdd)))
		events = append(events, evt)
	}
	return events, nil
}

// EventFee represent fee
type EventFee struct {
	TxID common.TxID
	Fee  common.Fee
}

// NewEventFee create a new EventFee
func NewEventFee(txID common.TxID, fee common.Fee) EventFee {
	return EventFee{
		TxID: txID,
		Fee:  fee,
	}
}

// Type get a string represent the event type
func (e EventFee) Type() string {
	return FeeEventType
}

// Events return events of cosmos.Event type
func (e EventFee) Events() (cosmos.Events, error) {
	evt := cosmos.NewEvent(e.Type(),
		cosmos.NewAttribute("tx_id", e.TxID.String()),
		cosmos.NewAttribute("coins", e.Fee.Coins.String()),
		cosmos.NewAttribute("pool_deduct", e.Fee.PoolDeduct.String()))
	return cosmos.Events{evt}, nil
}

// EventOutbound represent an outbound message from thornode
type EventOutbound struct {
	InTxID common.TxID // the inbound tx hash which triggered this outbound , it could be empty, because there are migration etc
	Tx     common.Tx
}

// NewEventOutbound create a new instance of EventOutbound
func NewEventOutbound(inTxID common.TxID, tx common.Tx) EventOutbound {
	return EventOutbound{
		InTxID: inTxID,
		Tx:     tx,
	}
}

// Type return a string which represent the type of this event
func (e EventOutbound) Type() string {
	return OutboundEventType
}

// Events return sdk events
func (e EventOutbound) Events() (cosmos.Events, error) {
	evt := cosmos.NewEvent(e.Type(),
		cosmos.NewAttribute("in_tx_id", e.InTxID.String()))
	evt = evt.AppendAttributes(e.Tx.ToAttributes()...)
	return cosmos.Events{evt}, nil
}
