package xdrill

import (
	"time"

	"github.com/stellar/go/xdr"
)

type TtlEntry struct {
	ttlEntry *xdr.TtlEntry
	change   *Change
}

func (t TtlEntry) LiveUntilLedgerSeq() uint32 {
	return uint32(t.ttlEntry.LiveUntilLedgerSeq)
}

func (t TtlEntry) LedgerKeyHash() string {
	return t.change.LedgerKeyHash()
}

func (t TtlEntry) Sponsor() string {
	return t.change.Sponsor()
}

func (t TtlEntry) LastModifiedLedger() uint32 {
	return t.change.LastModifiedLedger()
}

func (t TtlEntry) LedgerEntryChangeType() uint32 {
	return t.change.Type()
}

func (t TtlEntry) Deleted() bool {
	return t.change.Deleted()
}

func (t TtlEntry) ClosedAt() time.Time {
	return t.change.ClosedAt()
}

func (t TtlEntry) Sequence() uint32 {
	return t.change.Sequence()
}
