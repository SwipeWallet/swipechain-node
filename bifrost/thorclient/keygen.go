package thorclient

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	btypes "gitlab.com/thorchain/thornode/bifrost/blockscanner/types"
	"gitlab.com/thorchain/thornode/x/thorchain/types"
)

// GetKeygen retrieves keygen request for the given block height from thorchain
func (b *ThorchainBridge) GetKeygenBlock(blockHeight int64, pk string) (types.KeygenBlock, error) {
	path := fmt.Sprintf("%s/%d/%s", KeygenEndpoint, blockHeight, pk)
	body, status, err := b.getWithPath(path)
	if err != nil {
		if status == http.StatusNotFound {
			return types.KeygenBlock{}, btypes.UnavailableBlock
		}
		b.errCounter.WithLabelValues("fail_get_keygen", strconv.FormatInt(blockHeight, 10)).Inc()
		return types.KeygenBlock{}, fmt.Errorf("failed to get keygen for a block height: %w", err)
	}
	var query types.QueryKeygenBlock
	if err := b.cdc.UnmarshalJSON(body, &query); err != nil {
		b.errCounter.WithLabelValues("fail_unmarshal_keygen", strconv.FormatInt(blockHeight, 10)).Inc()
		return types.KeygenBlock{}, fmt.Errorf("failed to unmarshal Keygen: %w", err)
	}

	if query.Signature == "" {
		return types.KeygenBlock{}, errors.New("invalid keygen signature: empty")
	}

	buf, err := b.cdc.MarshalBinaryBare(query.KeygenBlock)
	if err != nil {
		return types.KeygenBlock{}, fmt.Errorf("fail to marshal keygen block to json: %w", err)
	}

	pubKey := b.keys.signerInfo.GetPubKey()
	s, err := base64.StdEncoding.DecodeString(query.Signature)
	if err != nil {
		return types.KeygenBlock{}, errors.New("invalid keygen signature: cannot decode signature")
	}
	if !pubKey.VerifyBytes(buf, s) {
		return types.KeygenBlock{}, errors.New("invalid keygen signature: bad signature")
	}

	return query.KeygenBlock, nil
}
