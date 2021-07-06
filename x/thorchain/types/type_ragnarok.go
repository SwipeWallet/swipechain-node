package types

import "gitlab.com/thorchain/thornode/common"

type RagnarokUnstakePosition struct {
	Number int64        `json:"number"`
	Pool   common.Asset `json:"pool"`
}

func (r RagnarokUnstakePosition) IsEmpty() bool {
	return r.Number < 0 || r.Pool.IsEmpty()
}
