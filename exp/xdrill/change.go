package xdrill

import (
	"fmt"
	"time"

	"github.com/stellar/go/exp/xdrill/utils"
	"github.com/stellar/go/ingest"
	"github.com/stellar/go/xdr"
)

type Change struct {
	change *ingest.Change
	ledger *Ledger
}

func (c Change) ExtractEntryFromChange() (xdr.LedgerEntry, xdr.LedgerEntryChangeType, bool, error) {
	switch changeType := c.change.LedgerEntryChangeType(); changeType {
	case xdr.LedgerEntryChangeTypeLedgerEntryCreated, xdr.LedgerEntryChangeTypeLedgerEntryUpdated:
		return *c.change.Post, changeType, false, nil
	case xdr.LedgerEntryChangeTypeLedgerEntryRemoved:
		return *c.change.Pre, changeType, true, nil
	default:
		return xdr.LedgerEntry{}, changeType, false, fmt.Errorf("unable to extract ledger entry type from change")
	}
}

func (c Change) Deleted() bool {
	_, _, deleted, err := c.ExtractEntryFromChange()
	if err != nil {
		panic(err)
	}

	return deleted
}

func (c Change) Type() uint32 {
	_, changeType, _, err := c.ExtractEntryFromChange()
	if err != nil {
		panic(err)
	}

	return uint32(changeType)
}

func (c Change) ClosedAt() time.Time {
	return c.ledger.ClosedAt()
}

func (c Change) Sequence() uint32 {
	return c.ledger.Sequence()
}

func (c Change) LastModifiedLedger() uint32 {
	ledgerEntry, _, _, err := c.ExtractEntryFromChange()
	if err != nil {
		panic(err)
	}

	return uint32(ledgerEntry.LastModifiedLedgerSeq)
}

func (c Change) Sponsor() string {
	ledgerEntry, _, _, err := c.ExtractEntryFromChange()
	if err != nil {
		panic(err)
	}

	if ledgerEntry.SponsoringID() == nil {
		return ""
	}

	return ledgerEntry.SponsoringID().Address()
}

func (c Change) LedgerKeyHash() string {
	ledgerEntry, _, _, err := c.ExtractEntryFromChange()
	if err != nil {
		panic(err)
	}

	return utils.LedgerEntryToLedgerKeyHash(ledgerEntry)
}
