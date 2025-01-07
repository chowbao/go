package xdrill

import (
	"time"

	"github.com/stellar/go/exp/xdrill/utils"
	"github.com/stellar/go/xdr"
)

type LiquidityPoolEntry struct {
	liquidityPoolEntry *xdr.LiquidityPoolEntry
	change             *Change
}

func (l LiquidityPoolEntry) LiquidityPoolID() string {
	return utils.PoolIDToString(l.liquidityPoolEntry.LiquidityPoolId)
}

func (l LiquidityPoolEntry) LiquidityPoolType() string {
	poolType, ok := xdr.LiquidityPoolTypeToString[l.liquidityPoolEntry.Body.Type]
	if !ok {
		return ""
	}

	return poolType
}

func (l LiquidityPoolEntry) LiquidityPoolFee() uint32 {
	constantProduct, ok := l.liquidityPoolEntry.Body.GetConstantProduct()
	if !ok {
		return 0
	}

	return uint32(constantProduct.Params.Fee)
}

func (l LiquidityPoolEntry) TrustlineCount() uint64 {
	constantProduct, ok := l.liquidityPoolEntry.Body.GetConstantProduct()
	if !ok {
		return 0
	}

	return uint64(constantProduct.PoolSharesTrustLineCount)
}

func (l LiquidityPoolEntry) PoolShareCount() float64 {
	constantProduct, ok := l.liquidityPoolEntry.Body.GetConstantProduct()
	if !ok {
		return 0
	}

	return utils.ConvertStroopValueToReal(constantProduct.TotalPoolShares)
}

func (l LiquidityPoolEntry) AssetACode() string {
	constantProduct, ok := l.liquidityPoolEntry.Body.GetConstantProduct()
	if !ok {
		return ""
	}

	asset, err := utils.TransformSingleAsset(constantProduct.Params.AssetA)
	if err != nil {
		panic(err)
	}

	return asset.AssetCode
}

func (l LiquidityPoolEntry) AssetAIssuer() string {
	constantProduct, ok := l.liquidityPoolEntry.Body.GetConstantProduct()
	if !ok {
		return ""
	}

	asset, err := utils.TransformSingleAsset(constantProduct.Params.AssetA)
	if err != nil {
		panic(err)
	}

	return asset.AssetIssuer
}

func (l LiquidityPoolEntry) AssetAType() string {
	constantProduct, ok := l.liquidityPoolEntry.Body.GetConstantProduct()
	if !ok {
		return ""
	}

	asset, err := utils.TransformSingleAsset(constantProduct.Params.AssetA)
	if err != nil {
		panic(err)
	}

	return asset.AssetType
}

func (l LiquidityPoolEntry) AssetAID() int64 {
	constantProduct, ok := l.liquidityPoolEntry.Body.GetConstantProduct()
	if !ok {
		return 0
	}

	asset, err := utils.TransformSingleAsset(constantProduct.Params.AssetA)
	if err != nil {
		panic(err)
	}

	return asset.AssetID
}

func (l LiquidityPoolEntry) AssetAReserve() float64 {
	constantProduct, ok := l.liquidityPoolEntry.Body.GetConstantProduct()
	if !ok {
		return 0
	}

	return utils.ConvertStroopValueToReal(constantProduct.ReserveA)
}

func (l LiquidityPoolEntry) AssetBIssuer() string {
	constantProduct, ok := l.liquidityPoolEntry.Body.GetConstantProduct()
	if !ok {
		return ""
	}

	asset, err := utils.TransformSingleAsset(constantProduct.Params.AssetB)
	if err != nil {
		panic(err)
	}

	return asset.AssetIssuer
}

func (l LiquidityPoolEntry) AssetBType() string {
	constantProduct, ok := l.liquidityPoolEntry.Body.GetConstantProduct()
	if !ok {
		return ""
	}

	asset, err := utils.TransformSingleAsset(constantProduct.Params.AssetB)
	if err != nil {
		panic(err)
	}

	return asset.AssetType
}

func (l LiquidityPoolEntry) AssetBID() int64 {
	constantProduct, ok := l.liquidityPoolEntry.Body.GetConstantProduct()
	if !ok {
		return 0
	}

	asset, err := utils.TransformSingleAsset(constantProduct.Params.AssetB)
	if err != nil {
		panic(err)
	}

	return asset.AssetID
}

func (l LiquidityPoolEntry) AssetBReserve() float64 {
	constantProduct, ok := l.liquidityPoolEntry.Body.GetConstantProduct()
	if !ok {
		return 0
	}

	return utils.ConvertStroopValueToReal(constantProduct.ReserveB)
}

func (l LiquidityPoolEntry) Sponsor() string {
	return l.change.Sponsor()
}

func (l LiquidityPoolEntry) LastModifiedLedger() uint32 {
	return l.change.LastModifiedLedger()
}

func (l LiquidityPoolEntry) LedgerEntryChangeType() uint32 {
	return l.change.Type()
}

func (l LiquidityPoolEntry) Deleted() bool {
	return l.change.Deleted()
}

func (l LiquidityPoolEntry) ClosedAt() time.Time {
	return l.change.ClosedAt()
}

func (l LiquidityPoolEntry) Sequence() uint32 {
	return l.change.Sequence()
}
