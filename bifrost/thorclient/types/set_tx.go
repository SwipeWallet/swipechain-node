package types

import (
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	cosmos "gitlab.com/thorchain/thornode/common/cosmos"
)

type SetTx struct {
	Mode string `json:"mode"`
	Tx   struct {
		Msg        []cosmos.Msg             `json:"msg"`
		Fee        authtypes.StdFee         `json:"fee"`
		Signatures []authtypes.StdSignature `json:"signatures"`
		Memo       string                   `json:"memo"`
	} `json:"tx"`
}
