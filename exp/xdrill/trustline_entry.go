package xdrill

import (
	"encoding/base64"
	"fmt"
	"time"

	"github.com/stellar/go/exp/xdrill/utils"
	"github.com/stellar/go/xdr"
)

type TrustlineEntry struct {
	trustlineEntry *xdr.TrustLineEntry
	change         *Change
}

func (t TrustlineEntry) LedgerKey() string {
	ledgerKey, err := t.trustLineEntryToLedgerKeyString()
	if err != nil {
		panic(err)
	}

	return ledgerKey
}

func (t TrustlineEntry) trustLineEntryToLedgerKeyString() (string, error) {
	ledgerKey := &xdr.LedgerKey{}
	err := ledgerKey.SetTrustline(t.trustlineEntry.AccountId, t.trustlineEntry.Asset)
	if err != nil {
		return "", fmt.Errorf("error running ledgerKey.SetTrustline when calculating ledger key")
	}

	key, err := ledgerKey.MarshalBinary()
	if err != nil {
		return "", fmt.Errorf("error running MarshalBinaryCompress when calculating ledger key")
	}

	return base64.StdEncoding.EncodeToString(key), nil

}

func (t TrustlineEntry) AccountID() string {
	return t.trustlineEntry.AccountId.Address()
}

func (t TrustlineEntry) AssetCode() string {
	asset, err := utils.TransformTrustLineAsset(t.trustlineEntry.Asset)
	if err != nil {
		panic(err)
	}

	return asset.AssetCode
}

func (t TrustlineEntry) AssetIssuer() string {
	asset, err := utils.TransformTrustLineAsset(t.trustlineEntry.Asset)
	if err != nil {
		panic(err)
	}

	return asset.AssetIssuer
}

func (t TrustlineEntry) AssetType() string {
	asset, err := utils.TransformTrustLineAsset(t.trustlineEntry.Asset)
	if err != nil {
		panic(err)
	}

	return asset.AssetType
}

func (t TrustlineEntry) AssetID() int64 {
	asset, err := utils.TransformTrustLineAsset(t.trustlineEntry.Asset)
	if err != nil {
		panic(err)
	}

	return asset.AssetID
}

func (t TrustlineEntry) Balance() float64 {
	return utils.ConvertStroopValueToReal(t.trustlineEntry.Balance)
}

func (t TrustlineEntry) TrustlineLimit() int64 {
	return int64(t.trustlineEntry.Limit)
}

func (t TrustlineEntry) LiquidityPoolID() string {
	if t.trustlineEntry.Asset.Type != xdr.AssetTypeAssetTypePoolShare {
		return ""
	}
	return utils.PoolIDToString(t.trustlineEntry.Asset.MustLiquidityPoolId())
}

func (t TrustlineEntry) BuyingLiabilities() float64 {
	return float64(t.trustlineEntry.Liabilities().Buying)
}

func (t TrustlineEntry) SellingLiabilities() float64 {
	return float64(t.trustlineEntry.Liabilities().Selling)
}

func (t TrustlineEntry) Flags() uint32 {
	return uint32(t.trustlineEntry.Flags)
}

func (t TrustlineEntry) Sponsor() string {
	return t.change.Sponsor()
}

func (t TrustlineEntry) LastModifiedLedger() uint32 {
	return t.change.LastModifiedLedger()
}

func (t TrustlineEntry) LedgerEntryChangeType() uint32 {
	return t.change.Type()
}

func (t TrustlineEntry) Deleted() bool {
	return t.change.Deleted()
}

func (t TrustlineEntry) ClosedAt() time.Time {
	return t.change.ClosedAt()
}

func (t TrustlineEntry) Sequence() uint32 {
	return t.change.Sequence()
}
