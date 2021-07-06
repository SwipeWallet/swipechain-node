package types

// AccountResp the response from thorclient
type AccountResp struct {
	Height string `json:"height"`
	Result struct {
		Value struct {
			AccountNumber uint64 `json:"account_number"`
			Sequence      uint64 `json:"sequence"`
		} `json:"value"`
	} `json:"result"`
}
