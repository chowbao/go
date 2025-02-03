package ledgerentry

import (
	"fmt"

	"github.com/stellar/go/xdr"
)

type ContractCode struct {
	ContractCodeHash  string `json:"contract_code_hash"`
	NInstructions     uint32 `json:"n_instructions"`
	NFunctions        uint32 `json:"n_functions"`
	NGlobals          uint32 `json:"n_globals"`
	NTableEntries     uint32 `json:"n_table_entries"`
	NTypes            uint32 `json:"n_types"`
	NDataSegments     uint32 `json:"n_data_segments"`
	NElemSegments     uint32 `json:"n_elem_segments"`
	NImports          uint32 `json:"n_imports"`
	NExports          uint32 `json:"n_exports"`
	NDataSegmentBytes uint32 `json:"n_data_segment_bytes"`
}

func ContractCodeDetails(contractCodeEntry *xdr.ContractCodeEntry) (ContractCode, error) {
	var contractCode ContractCode

	switch contractCodeEntry.Ext.V {
	case 1:
		contractCode.NInstructions = uint32(contractCodeEntry.Ext.V1.CostInputs.NInstructions)
		contractCode.NFunctions = uint32(contractCodeEntry.Ext.V1.CostInputs.NFunctions)
		contractCode.NGlobals = uint32(contractCodeEntry.Ext.V1.CostInputs.NGlobals)
		contractCode.NTableEntries = uint32(contractCodeEntry.Ext.V1.CostInputs.NTableEntries)
		contractCode.NTypes = uint32(contractCodeEntry.Ext.V1.CostInputs.NTypes)
		contractCode.NDataSegments = uint32(contractCodeEntry.Ext.V1.CostInputs.NDataSegments)
		contractCode.NElemSegments = uint32(contractCodeEntry.Ext.V1.CostInputs.NElemSegments)
		contractCode.NImports = uint32(contractCodeEntry.Ext.V1.CostInputs.NImports)
		contractCode.NExports = uint32(contractCodeEntry.Ext.V1.CostInputs.NExports)
		contractCode.NDataSegmentBytes = uint32(contractCodeEntry.Ext.V1.CostInputs.NDataSegmentBytes)
	default:
		return ContractCode{}, fmt.Errorf("unknown ContractCodeEntry.Ext.V")
	}

	var err error
	var contractCodeHash string
	contractCodeHash, err = contractCodeEntry.Hash.MarshalBinaryBase64()
	if err != nil {
		return ContractCode{}, err
	}

	contractCode.ContractCodeHash = contractCodeHash

	return contractCode, nil
}
