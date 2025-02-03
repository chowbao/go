package ledgerentry

import (
	"github.com/stellar/go/xdr"
)

type Account struct {
	AccountID            string    `json:"account_id"`
	Balance              int64     `json:"balance"`
	SequenceNumber       int64     `json:"sequence_number"`
	SequenceLedger       uint32    `json:"sequence_ledger"`
	SequenceTime         int64     `json:"sequence_time"`
	NumSubentries        uint32    `json:"num_subentries"`
	Flags                uint32    `json:"flags"`
	HomeDomain           string    `json:"home_domain"`
	MasterWeight         int32     `json:"master_weight"`
	ThresholdLow         int32     `json:"threshold_low"`
	ThresholdMedium      int32     `json:"threshold_medium"`
	ThresholdHigh        int32     `json:"threshold_high"`
	NumSponsored         uint32    `json:"num_sponsored"`
	NumSponsoring        uint32    `json:"num_sponsoring"`
	BuyingLiabilities    int64     `json:"buying_liabilities"`
	SellingLiabilities   int64     `json:"selling_liabilities"`
	InflationDestination string    `json:"inflation_destination"`
	Signers              []Signers `json:"signers"`
}

type Signers struct {
	Address string
	Weight  int32
	Sponsor string
}

func AccountDetails(accountEntry *xdr.AccountEntry) (Account, error) {
	account := Account{
		AccountID:       accountEntry.AccountId.Address(),
		SequenceNumber:  int64(accountEntry.SeqNum),
		SequenceLedger:  uint32(accountEntry.SeqLedger()),
		SequenceTime:    int64(accountEntry.SeqTime()),
		NumSubentries:   uint32(accountEntry.NumSubEntries),
		Flags:           uint32(accountEntry.Flags),
		HomeDomain:      string(accountEntry.HomeDomain),
		MasterWeight:    int32(accountEntry.MasterKeyWeight()),
		ThresholdLow:    int32(accountEntry.ThresholdLow()),
		ThresholdMedium: int32(accountEntry.ThresholdMedium()),
		ThresholdHigh:   int32(accountEntry.ThresholdHigh()),
		NumSponsored:    uint32(accountEntry.NumSponsored()),
		NumSponsoring:   uint32(accountEntry.NumSponsoring()),
	}

	if accountEntry.InflationDest != nil {
		account.InflationDestination = accountEntry.InflationDest.Address()
	}

	accountExtensionInfo, ok := accountEntry.Ext.GetV1()
	if ok {
		account.BuyingLiabilities = int64(accountExtensionInfo.Liabilities.Buying)
		account.SellingLiabilities = int64(accountExtensionInfo.Liabilities.Selling)
	}

	signers := []Signers{}
	sponsors := accountEntry.SponsorPerSigner()
	for signer, weight := range accountEntry.SignerSummary() {
		sponsorDesc := sponsors[signer]

		signers = append(signers, Signers{
			Address: signer,
			Weight:  weight,
			Sponsor: sponsorDesc.Address(),
		})
	}

	account.Signers = signers

	return account, nil
}
