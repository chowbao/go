package xdrill

import (
	"time"

	"github.com/stellar/go/exp/xdrill/utils"
	"github.com/stellar/go/xdr"
)

type AccountEntry struct {
	accountEntry *xdr.AccountEntry
	change       *Change
}

func (a AccountEntry) AccountID() string {
	return a.accountEntry.AccountId.Address()
}

func (a AccountEntry) Balance() float64 {
	return utils.ConvertStroopValueToReal(a.accountEntry.Balance)
}

func (a AccountEntry) BuyingLiabilities() float64 {
	var buyingLiabilities float64
	accountExtensionInfo, V1Found := a.accountEntry.Ext.GetV1()
	if V1Found {
		return utils.ConvertStroopValueToReal(accountExtensionInfo.Liabilities.Buying)
	}
	return buyingLiabilities
}

func (a AccountEntry) SellingLiabilities() float64 {
	var sellingLiabilities float64
	accountExtensionInfo, V1Found := a.accountEntry.Ext.GetV1()
	if V1Found {
		return utils.ConvertStroopValueToReal(accountExtensionInfo.Liabilities.Selling)
	}
	return sellingLiabilities
}

func (a AccountEntry) SequenceNumber() int64 {
	return int64(a.accountEntry.SeqNum)
}

func (a AccountEntry) SequenceLedger() int64 {
	return int64(a.accountEntry.SeqLedger())
}

func (a AccountEntry) SequenceTime() int64 {
	return int64(a.accountEntry.SeqTime())
}

func (a AccountEntry) NumSubEntries() uint32 {
	return uint32(a.accountEntry.NumSubEntries)
}

func (a AccountEntry) InflationDestination() string {
	var inflationDest string
	inflationDestAccountID := a.accountEntry.InflationDest
	if inflationDestAccountID != nil {
		return inflationDestAccountID.Address()
	}
	return inflationDest
}

func (a AccountEntry) Flags() uint32 {
	return uint32(a.accountEntry.Flags)
}

func (a AccountEntry) HomeDomain() string {
	return string(a.accountEntry.HomeDomain)
}

func (a AccountEntry) MasterKeyWeight() int32 {
	return int32(a.accountEntry.MasterKeyWeight())
}

func (a AccountEntry) ThresholdLow() int32 {
	return int32(a.accountEntry.ThresholdLow())
}

func (a AccountEntry) ThresholdMedium() int32 {
	return int32(a.accountEntry.ThresholdMedium())
}

func (a AccountEntry) ThresholdHigh() int32 {
	return int32(a.accountEntry.ThresholdHigh())
}

func (a AccountEntry) Sponsor() string {
	return a.change.Sponsor()
}

func (a AccountEntry) NumSponsored() uint32 {
	return uint32(a.accountEntry.NumSponsored())
}

func (a AccountEntry) NumSponsoring() uint32 {
	return uint32(a.accountEntry.NumSponsoring())
}

func (a AccountEntry) LastModifiedLedger() uint32 {
	return a.change.LastModifiedLedger()
}

func (a AccountEntry) LedgerEntryChangeType() uint32 {
	return a.change.Type()
}

func (a AccountEntry) Deleted() bool {
	return a.change.Deleted()
}

func (a AccountEntry) ClosedAt() time.Time {
	return a.change.ClosedAt()
}

func (a AccountEntry) Sequence() uint32 {
	return a.change.Sequence()
}

type Signers struct {
	Address string
	Weight  int32
	Sponsor string
}

func (a AccountEntry) Signers() []Signers {
	ledgerEntry, _, _, err := a.change.ExtractEntryFromChange()
	if err != nil {
		panic(err)
	}

	accountEntry := ledgerEntry.Data.MustAccount()

	signers := []Signers{}
	sponsors := accountEntry.SponsorPerSigner()
	for signer, weight := range accountEntry.SignerSummary() {
		sponsorDesc, _ := sponsors[signer]

		signers = append(signers, Signers{
			Address: signer,
			Weight:  weight,
			Sponsor: sponsorDesc.Address(),
		})
	}

	return signers
}
