package ledgerentry

import (
	"github.com/stellar/go/xdr"
)

type ClaimableBalance struct {
	BalanceID   string     `json:"balance_id"`
	Claimants   []Claimant `json:"claimants"`
	AssetCode   string     `json:"asset_code"`
	AssetIssuer string     `json:"asset_issuer"`
	AssetType   string     `json:"asset_type"`
	AssetID     int64      `json:"asset_id"`
	Amount      int64      `json:"amount"`
	Flags       uint32     `json:"flags"`
}

type Claimant struct {
	Destination string             `json:"destination"`
	Predicate   xdr.ClaimPredicate `json:"predicate"`
}

func ClaimableBalanceDetails(claimableBalanceEntry *xdr.ClaimableBalanceEntry) (ClaimableBalance, error) {
	claimableBalance := ClaimableBalance{
		Amount: int64(claimableBalanceEntry.Amount),
		Flags:  uint32(claimableBalanceEntry.Flags()),
	}

	var err error
	var balanceID string
	balanceID, err = claimableBalanceEntry.BalanceId.MarshalBinaryBase64()
	if err != nil {
		return ClaimableBalance{}, err
	}

	claimableBalance.BalanceID = balanceID

	var assetType, assetCode, assetIssuer string
	err = claimableBalanceEntry.Asset.Extract(&assetType, &assetCode, &assetIssuer)
	if err != nil {
		return ClaimableBalance{}, err
	}

	claimableBalance.AssetCode = assetCode
	claimableBalance.AssetIssuer = assetIssuer
	claimableBalance.AssetType = assetType

	var claimants []Claimant
	for _, c := range claimableBalanceEntry.Claimants {
		switch c.Type {
		case 0:
			claimants = append(claimants, Claimant{
				Destination: c.V0.Destination.Address(),
				Predicate:   c.V0.Predicate,
			})
		}
	}

	claimableBalance.Claimants = claimants

	return claimableBalance, nil
}
