package transactions

import (
	"github.com/stellar/go/ingest"
)

type Transactions struct {
	ingest.LedgerTransaction
}
