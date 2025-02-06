package ledgerentry

import "github.com/stellar/go/xdr"

type Ttl struct {
	LiveUntilLedgerSeq uint32
}

func TtlDetails(ttlEntry *xdr.TtlEntry) (Ttl, error) {
	return Ttl{
		LiveUntilLedgerSeq: uint32(ttlEntry.LiveUntilLedgerSeq),
	}, nil
}
