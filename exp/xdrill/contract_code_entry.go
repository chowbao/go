package xdrill

import (
	"time"

	"github.com/stellar/go/xdr"
)

type ContractCodeEntry struct {
	contractCodeEntry *xdr.ContractCodeEntry
	change            *Change
}

func (c ContractCodeEntry) ContractCodeHash() string {
	return c.contractCodeEntry.Hash.HexString()
}

func (c ContractCodeEntry) LedgerKeyHash() string {
	return c.change.LedgerKeyHash()
}

func (c ContractCodeEntry) NInstructions() uint32 {
	switch c.contractCodeEntry.Ext.V {
	case 1:
		return uint32(c.contractCodeEntry.Ext.V1.CostInputs.NInstructions)
	}

	return 0
}

func (c ContractCodeEntry) NFunctions() uint32 {
	switch c.contractCodeEntry.Ext.V {
	case 1:
		return uint32(c.contractCodeEntry.Ext.V1.CostInputs.NFunctions)
	}

	return 0
}

func (c ContractCodeEntry) NGlobals() uint32 {
	switch c.contractCodeEntry.Ext.V {
	case 1:
		return uint32(c.contractCodeEntry.Ext.V1.CostInputs.NGlobals)
	}

	return 0
}

func (c ContractCodeEntry) NTableEntries() uint32 {
	switch c.contractCodeEntry.Ext.V {
	case 1:
		return uint32(c.contractCodeEntry.Ext.V1.CostInputs.NTableEntries)
	}

	return 0
}

func (c ContractCodeEntry) NTypes() uint32 {
	switch c.contractCodeEntry.Ext.V {
	case 1:
		return uint32(c.contractCodeEntry.Ext.V1.CostInputs.NTypes)
	}

	return 0
}

func (c ContractCodeEntry) NDataSegments() uint32 {
	switch c.contractCodeEntry.Ext.V {
	case 1:
		return uint32(c.contractCodeEntry.Ext.V1.CostInputs.NDataSegments)
	}

	return 0
}

func (c ContractCodeEntry) NElemSegments() uint32 {
	switch c.contractCodeEntry.Ext.V {
	case 1:
		return uint32(c.contractCodeEntry.Ext.V1.CostInputs.NElemSegments)
	}

	return 0
}

func (c ContractCodeEntry) NImports() uint32 {
	switch c.contractCodeEntry.Ext.V {
	case 1:
		return uint32(c.contractCodeEntry.Ext.V1.CostInputs.NImports)
	}

	return 0
}

func (c ContractCodeEntry) NExports() uint32 {
	switch c.contractCodeEntry.Ext.V {
	case 1:
		return uint32(c.contractCodeEntry.Ext.V1.CostInputs.NExports)
	}

	return 0
}

func (c ContractCodeEntry) NDataSegmentBytes() uint32 {
	switch c.contractCodeEntry.Ext.V {
	case 1:
		return uint32(c.contractCodeEntry.Ext.V1.CostInputs.NDataSegmentBytes)
	}

	return 0
}

func (c ContractCodeEntry) Sponsor() string {
	return c.change.Sponsor()
}

func (c ContractCodeEntry) LastModifiedLedger() uint32 {
	return c.change.LastModifiedLedger()
}

func (c ContractCodeEntry) LedgerEntryChangeType() uint32 {
	return c.change.Type()
}

func (c ContractCodeEntry) Deleted() bool {
	return c.change.Deleted()
}

func (c ContractCodeEntry) ClosedAt() time.Time {
	return c.change.ClosedAt()
}

func (c ContractCodeEntry) Sequence() uint32 {
	return c.change.Sequence()
}
