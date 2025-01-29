package xdr

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"

	"github.com/stellar/go/support/errors"
)

func NewPoolId(a, b Asset, fee Int32) (PoolId, error) {
	if b.LessThan(a) {
		return PoolId{}, errors.New("AssetA must be < AssetB")
	}

	// Assume the assets are already sorted.
	params := LiquidityPoolParameters{
		Type: LiquidityPoolTypeLiquidityPoolConstantProduct,
		ConstantProduct: &LiquidityPoolConstantProductParameters{
			AssetA: a,
			AssetB: b,
			Fee:    fee,
		},
	}

	buf := &bytes.Buffer{}
	if _, err := Marshal(buf, params); err != nil {
		return PoolId{}, errors.Wrap(err, "failed to build liquidity pool id")
	}
	return sha256.Sum256(buf.Bytes()), nil
}

// MarshalBinaryBase64 marshals XDR into a binary form and then encodes it
// using base64.
func (p PoolId) MarshalBinaryBase64() (string, error) {
	b, err := p.MarshalBinary()
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(b), nil
}
