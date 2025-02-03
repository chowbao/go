package ledgerentry

import "github.com/stellar/go/xdr"

type Trustline struct {
	AccountID          string `json:"account_id"`
	AssetCode          string `json:"asset_code"`
	AssetIssuer        string `json:"asset_issuer"`
	AssetType          string `json:"asset_type"`
	Balance            int64  `json:"balance"`
	TrustlineLimit     int64  `json:"trustline_limit"`
	LiquidityPoolID    string `json:"liquidity_pool_id"`
	BuyingLiabilities  int64  `json:"buying_liabilities"`
	SellingLiabilities int64  `json:"selling_liabilities"`
	Flags              uint32 `json:"flags"`
}

func TrustlineDetails(trustlineEntry *xdr.TrustLineEntry) (Trustline, error) {
	trustline := Trustline{
		AccountID:          trustlineEntry.AccountId.Address(),
		Balance:            int64(trustlineEntry.Balance),
		TrustlineLimit:     int64(trustlineEntry.Limit),
		BuyingLiabilities:  int64(trustlineEntry.Liabilities().Buying),
		SellingLiabilities: int64(trustlineEntry.Liabilities().Selling),
		Flags:              uint32(trustlineEntry.Flags),
	}

	var err error
	var assetType, assetCode, assetIssuer string
	err = trustlineEntry.Asset.Extract(&assetType, &assetCode, &assetIssuer)
	if err != nil {
		return Trustline{}, err
	}

	trustline.AssetCode = assetCode
	trustline.AssetIssuer = assetIssuer
	trustline.AssetType = assetType

	var poolID string
	poolID, err = xdr.MarshalBase64(trustlineEntry.Asset.LiquidityPoolId)
	if err != nil {
		return Trustline{}, err
	}

	trustline.LiquidityPoolID = poolID

	return trustline, nil
}
