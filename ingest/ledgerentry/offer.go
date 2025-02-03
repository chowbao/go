package ledgerentry

import "github.com/stellar/go/xdr"

type Offer struct {
	SellerID           string `json:"seller_id"`
	OfferID            int64  `json:"offer_id"`
	SellingAssetType   string `json:"selling_asset_type"`
	SellingAssetCode   string `json:"selling_asset_code"`
	SellingAssetIssuer string `json:"selling_asset_issuer"`
	BuyingAssetType    string `json:"buying_asset_type"`
	BuyingAssetCode    string `json:"buying_asset_code"`
	BuyingAssetIssuer  string `json:"buying_asset_issuer"`
	Amount             int64  `json:"amount"`
	PriceN             int32  `json:"pricen"`
	PriceD             int32  `json:"priced"`
	Flags              uint32 `json:"flags"`
}

func OfferDetails(offerEntry *xdr.OfferEntry) (Offer, error) {
	offer := Offer{
		SellerID: offerEntry.SellerId.Address(),
		OfferID:  int64(offerEntry.OfferId),
		Amount:   int64(offerEntry.Amount),
		PriceN:   int32(offerEntry.Price.N),
		PriceD:   int32(offerEntry.Price.D),
		Flags:    uint32(offerEntry.Flags),
	}

	var err error
	var sellingAssetType, sellingAssetCode, sellingAssetIssuer string
	err = offerEntry.Selling.Extract(&sellingAssetType, &sellingAssetCode, &sellingAssetIssuer)
	if err != nil {
		return Offer{}, err
	}

	offer.SellingAssetCode = sellingAssetCode
	offer.SellingAssetIssuer = sellingAssetIssuer
	offer.SellingAssetType = sellingAssetType

	var buyingAssetType, buyingAssetCode, buyingAssetIssuer string
	err = offerEntry.Buying.Extract(&buyingAssetType, &buyingAssetCode, &buyingAssetIssuer)
	if err != nil {
		return Offer{}, err
	}

	offer.BuyingAssetCode = buyingAssetCode
	offer.BuyingAssetIssuer = buyingAssetIssuer
	offer.BuyingAssetType = buyingAssetType

	return offer, nil
}
