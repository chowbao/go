package xdrill

import (
	"time"

	"github.com/stellar/go/exp/xdrill/utils"
	"github.com/stellar/go/xdr"
)

type OfferEntry struct {
	offerEntry *xdr.OfferEntry
	change     *Change
}

func (o OfferEntry) SellerID() string {
	return o.offerEntry.SellerId.Address()
}

func (o OfferEntry) OfferID() int64 {
	return int64(o.offerEntry.OfferId)
}

func (o OfferEntry) SellingAssetCode() string {
	asset, err := utils.TransformSingleAsset(o.offerEntry.Selling)
	if err != nil {
		panic(err)
	}

	return asset.AssetCode
}

func (o OfferEntry) SellingAssetIssuer() string {
	asset, err := utils.TransformSingleAsset(o.offerEntry.Selling)
	if err != nil {
		panic(err)
	}

	return asset.AssetIssuer
}

func (o OfferEntry) SellingAssetType() string {
	asset, err := utils.TransformSingleAsset(o.offerEntry.Selling)
	if err != nil {
		panic(err)
	}

	return asset.AssetType
}

func (o OfferEntry) SellingAssetID() int64 {
	asset, err := utils.TransformSingleAsset(o.offerEntry.Selling)
	if err != nil {
		panic(err)
	}

	return asset.AssetID
}

func (o OfferEntry) BuyingAssetCode() string {
	asset, err := utils.TransformSingleAsset(o.offerEntry.Buying)
	if err != nil {
		panic(err)
	}

	return asset.AssetCode
}

func (o OfferEntry) BuyingAssetIssuer() string {
	asset, err := utils.TransformSingleAsset(o.offerEntry.Buying)
	if err != nil {
		panic(err)
	}

	return asset.AssetIssuer
}

func (o OfferEntry) BuyingAssetType() string {
	asset, err := utils.TransformSingleAsset(o.offerEntry.Buying)
	if err != nil {
		panic(err)
	}

	return asset.AssetType
}

func (o OfferEntry) BuyingAssetID() int64 {
	asset, err := utils.TransformSingleAsset(o.offerEntry.Buying)
	if err != nil {
		panic(err)
	}

	return asset.AssetID
}

func (o OfferEntry) Amount() float64 {
	return utils.ConvertStroopValueToReal(o.offerEntry.Amount)
}

func (o OfferEntry) PriceN() int32 {
	return int32(o.offerEntry.Price.N)
}

func (o OfferEntry) PriceD() int32 {
	return int32(o.offerEntry.Price.D)
}

func (o OfferEntry) Price() float64 {
	return float64(o.PriceN()) / float64(o.PriceD())
}

func (o OfferEntry) Flags() uint32 {
	return uint32(o.offerEntry.Flags)
}

func (o OfferEntry) Sponsor() string {
	return o.change.Sponsor()
}

func (o OfferEntry) LastModifiedLedger() uint32 {
	return o.change.LastModifiedLedger()
}

func (o OfferEntry) LedgerEntryChangeType() uint32 {
	return o.change.Type()
}

func (o OfferEntry) Deleted() bool {
	return o.change.Deleted()
}

func (o OfferEntry) ClosedAt() time.Time {
	return o.change.ClosedAt()
}

func (o OfferEntry) Sequence() uint32 {
	return o.change.Sequence()
}
