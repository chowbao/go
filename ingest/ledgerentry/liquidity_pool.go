package ledgerentry

import (
	"github.com/stellar/go/xdr"
)

type LiquidityPool struct {
	LiquidityPoolID string `json:"liquidity_pool_id"`
	Type            int32  `json:"type"`
	Fee             int32  `json:"fee"`
	TrustlineCount  int64  `json:"trustline_count"`
	PoolShareCount  int64  `json:"pool_share_count"`
	AssetAType      string `json:"asset_a_type"`
	AssetACode      string `json:"asset_a_code"`
	AssetAIssuer    string `json:"asset_a_issuer"`
	AssetAReserve   int64  `json:"asset_a_amount"`
	AssetBType      string `json:"asset_b_type"`
	AssetBCode      string `json:"asset_b_code"`
	AssetBIssuer    string `json:"asset_b_issuer"`
	AssetBReserve   int64  `json:"asset_b_amount"`
}

func LiquidityPoolDetails(liquidityPoolEntry *xdr.LiquidityPoolEntry) (LiquidityPool, error) {
	lp := LiquidityPool{
		Type: int32(liquidityPoolEntry.Body.Type),
	}

	var err error
	var liquidtiyPoolID string
	liquidtiyPoolID, err = xdr.MarshalBase64(liquidityPoolEntry.LiquidityPoolId)
	if err != nil {
		return LiquidityPool{}, err
	}

	lp.LiquidityPoolID = liquidtiyPoolID

	var ok bool
	var constantProduct xdr.LiquidityPoolEntryConstantProduct
	constantProduct, ok = liquidityPoolEntry.Body.GetConstantProduct()
	if !ok {
		return lp, nil
	}

	lp.Fee = int32(constantProduct.Params.Fee)
	lp.TrustlineCount = int64(constantProduct.PoolSharesTrustLineCount)
	lp.PoolShareCount = int64(constantProduct.TotalPoolShares)

	var assetAType, assetACode, assetAIssuer string
	err = constantProduct.Params.AssetA.Extract(&assetAType, &assetACode, &assetAIssuer)
	if err != nil {
		return LiquidityPool{}, err
	}

	lp.AssetACode = assetACode
	lp.AssetAIssuer = assetAIssuer
	lp.AssetAType = assetAType
	lp.AssetAReserve = int64(constantProduct.ReserveA)

	var assetBType, assetBCode, assetBIssuer string
	err = constantProduct.Params.AssetB.Extract(&assetBType, &assetBCode, &assetBIssuer)
	if err != nil {
		return LiquidityPool{}, err
	}

	lp.AssetBCode = assetBCode
	lp.AssetBIssuer = assetBIssuer
	lp.AssetBType = assetBType
	lp.AssetBReserve = int64(constantProduct.ReserveB)

	return lp, nil
}
