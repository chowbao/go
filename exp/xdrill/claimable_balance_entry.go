package xdrill

import (
	"time"

	"github.com/stellar/go/exp/xdrill/utils"
	"github.com/stellar/go/xdr"
)

type ClaimableBalanceEntry struct {
	claimableBalanceEntry *xdr.ClaimableBalanceEntry
	change                *Change
}

func (c ClaimableBalanceEntry) BalanceID() string {
	balanceID, err := xdr.MarshalHex(c.claimableBalanceEntry.BalanceId)
	if err != nil {
		panic(err)
	}

	return balanceID
}

func (c ClaimableBalanceEntry) AssetCode() string {
	asset, err := utils.TransformSingleAsset(c.claimableBalanceEntry.Asset)
	if err != nil {
		panic(err)
	}

	return asset.AssetCode
}

func (c ClaimableBalanceEntry) AssetIssuer() string {
	asset, err := utils.TransformSingleAsset(c.claimableBalanceEntry.Asset)
	if err != nil {
		panic(err)
	}

	return asset.AssetIssuer
}

func (c ClaimableBalanceEntry) AssetType() string {
	asset, err := utils.TransformSingleAsset(c.claimableBalanceEntry.Asset)
	if err != nil {
		panic(err)
	}

	return asset.AssetType
}

func (c ClaimableBalanceEntry) AssetID() int64 {
	asset, err := utils.TransformSingleAsset(c.claimableBalanceEntry.Asset)
	if err != nil {
		panic(err)
	}

	return asset.AssetID
}

func (c ClaimableBalanceEntry) Claimants() []utils.Claimant {
	return utils.TransformClaimants(c.claimableBalanceEntry.Claimants)
}

func (c ClaimableBalanceEntry) Amount() float64 {
	return float64(c.claimableBalanceEntry.Amount) / 1.0e7
}

func (c ClaimableBalanceEntry) Flags() uint32 {
	return uint32(c.claimableBalanceEntry.Flags())
}

func (c ClaimableBalanceEntry) Sponsor() string {
	return c.change.Sponsor()
}

func (c ClaimableBalanceEntry) LastModifiedLedger() uint32 {
	return c.change.LastModifiedLedger()
}

func (c ClaimableBalanceEntry) LedgerEntryChangeType() uint32 {
	return c.change.Type()
}

func (c ClaimableBalanceEntry) Deleted() bool {
	return c.change.Deleted()
}

func (c ClaimableBalanceEntry) ClosedAt() time.Time {
	return c.change.ClosedAt()
}

func (c ClaimableBalanceEntry) Sequence() uint32 {
	return c.change.Sequence()
}
