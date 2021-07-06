package thorchain

import (
	"fmt"

	"gitlab.com/thorchain/thornode/common/cosmos"
)

// EmitEventItem define the method all event need to implement
type EmitEventItem interface {
	Events() (cosmos.Events, error)
}

// EventMgrV1 implement EventManager interface
type EventMgrV1 struct{}

// NewEventMgrV1 create a new instance of EventMgrV1
func NewEventMgrV1() *EventMgrV1 {
	return &EventMgrV1{}
}

// EmitEvent to block
func (m *EventMgrV1) EmitEvent(ctx cosmos.Context, evt EmitEventItem) error {
	events, err := evt.Events()
	if err != nil {
		return fmt.Errorf("fail to get events: %w", err)
	}
	ctx.EventManager().EmitEvents(events)
	return nil
}

// EmitGasEvent emit gas events
func (m *EventMgrV1) EmitGasEvent(ctx cosmos.Context, gasEvent *EventGas) error {
	if gasEvent == nil {
		return nil
	}
	return m.EmitEvent(ctx, gasEvent)
}

// EmitSwapEvent emit swap event to block
func (m *EventMgrV1) EmitSwapEvent(ctx cosmos.Context, swap EventSwap) error {
	// OutTxs is a temporary field that we used, as for now we need to keep backward compatibility so the
	// events change doesn't break midgard and smoke test, for double swap , we first swap the source asset to RUNE ,
	// and then from RUNE to target asset, so the first will be marked as success
	if !swap.OutTxs.IsEmpty() {
		outboundEvt := NewEventOutbound(swap.InTx.ID, swap.OutTxs)
		if err := m.EmitEvent(ctx, outboundEvt); err != nil {
			return fmt.Errorf("fail to emit an outbound event for double swap: %w", err)
		}
	}
	return m.EmitEvent(ctx, swap)
}

// EmitFeeEvent emit a fee event through event manager
func (m *EventMgrV1) EmitFeeEvent(ctx cosmos.Context, feeEvent EventFee) error {
	if feeEvent.Fee.Coins.IsEmpty() && feeEvent.Fee.PoolDeduct.IsZero() {
		return nil
	}

	if feeEvent.Fee.Coins.IsEmpty() && feeEvent.Fee.PoolDeduct.IsZero() {
		return nil
	}
	events, err := feeEvent.Events()
	if err != nil {
		return fmt.Errorf("fail to emit fee event: %w", err)
	}
	ctx.EventManager().EmitEvents(events)
	return nil
}
