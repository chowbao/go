package main

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"math"

	"github.com/stellar/go/ingest"
	"github.com/stellar/go/ingest/ledgerbackend"
	"github.com/stellar/go/xdr"
)

func StreamChanges(backend *ledgerbackend.LedgerBackend, start, end, batchSize uint32, changeChannel chan ChangeBatch, closeChan chan int) {
	batchStart := start
	batchEnd := uint32(math.Min(float64(batchStart+batchSize), float64(end)))
	for batchStart < batchEnd {
		if batchEnd < end {
			batchEnd = uint32(batchEnd - 1)
		}
		batch := ExtractBatch(batchStart, batchEnd, backend)
		changeChannel <- batch
		// batchStart and batchEnd should not overlap
		// overlapping batches causes duplicate record loads
		batchStart = uint32(math.Min(float64(batchEnd), float64(end)) + 1)
		batchEnd = uint32(math.Min(float64(batchStart+batchSize), float64(end)))
	}
	close(changeChannel)
	closeChan <- 1
}

type LedgerChanges struct {
	Changes       []ingest.Change
	LedgerHeaders []xdr.LedgerHeaderHistoryEntry
}

type ChangeBatch struct {
	Changes    map[xdr.LedgerEntryType]LedgerChanges
	BatchStart uint32
	BatchEnd   uint32
}

func LedgerStreamChanges(backend *ledgerbackend.LedgerBackend, start, end uint32, contracts []string) {
	changeChan := make(chan ChangeBatch)
	closeChan := make(chan int)

	go StreamChanges(backend, start, end, 1, changeChan, closeChan)

	for {
		select {
		case <-closeChan:
			return
		case batch, ok := <-changeChan:
			if !ok {
				continue
			}
			transformedOutputs := map[string][]interface{}{
				"accounts":           {},
				"signers":            {},
				"claimable_balances": {},
				"offers":             {},
				"trustlines":         {},
				"liquidity_pools":    {},
				"contract_data":      {},
				"contract_code":      {},
				"config_settings":    {},
				"ttl":                {},
			}

			for entryType, changes := range batch.Changes {
				switch entryType {
				case xdr.LedgerEntryTypeContractData:
					for i, change := range changes.Changes {
						TransformContractData := NewTransformContractDataStruct(AssetFromContractData, ContractBalanceFromContractData)
						contractData, err, _ := TransformContractData.TransformContractData(change, "Public Global Stellar Network ; September 2015", changes.LedgerHeaders[i])
						if err != nil {
							continue
						}

						// Empty contract data that has no error is a nonce. Does not need to be recorded
						if contractData.ContractId == "" {
							continue
						}

						transformedOutputs["contract_data"] = append(transformedOutputs["contract_data"], contractData)

						db, _ := sql.Open("duckdb", "index_database.duckdb")
						defer db.Close()

						query := `
							INSERT INTO contract_data_full (
								contract_id, contract_key_type, contract_durability, asset_code, 
								asset_issuer, asset_type, balance_holder, balance, 
								last_modified_ledger, ledger_entry_change, deleted, closed_at, 
								ledger_sequence, ledger_key_hash
							) 
							select ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
							where not exists (
								select 1 from contract_data_full where contract_id = ? and ledger_sequence = ? and ledger_key_hash = ? and ledger_entry_change = ?
							);
						`

						for _, contract := range contracts {
							if contract == contractData.ContractId {
								db.Exec(query,
									contractData.ContractId,
									contractData.ContractKeyType,
									contractData.ContractDurability,
									contractData.ContractDataAssetCode,
									contractData.ContractDataAssetIssuer,
									contractData.ContractDataAssetType,
									contractData.ContractDataBalanceHolder,
									contractData.ContractDataBalance,
									contractData.LastModifiedLedger,
									contractData.LedgerEntryChange,
									contractData.Deleted,
									contractData.ClosedAt,
									contractData.LedgerSequence,
									contractData.LedgerKeyHash,
									contractData.ContractId,
									contractData.LedgerSequence,
									contractData.LedgerKeyHash,
									contractData.LedgerEntryChange,
								)

								fmt.Printf("ledger: %d; contract_id: %s; key_hash: %s\n", contractData.LedgerSequence, contractData.ContractId, contractData.LedgerKeyHash)
							}
						}

					}
				}
			}
		}
	}
}

// extractBatch gets the changes from the ledgers in the range [batchStart, batchEnd] and compacts them
func ExtractBatch(
	batchStart, batchEnd uint32,
	backend *ledgerbackend.LedgerBackend) ChangeBatch {

	dataTypes := []xdr.LedgerEntryType{
		xdr.LedgerEntryTypeAccount,
		xdr.LedgerEntryTypeOffer,
		xdr.LedgerEntryTypeTrustline,
		xdr.LedgerEntryTypeLiquidityPool,
		xdr.LedgerEntryTypeClaimableBalance,
		xdr.LedgerEntryTypeContractData,
		xdr.LedgerEntryTypeContractCode,
		xdr.LedgerEntryTypeConfigSetting,
		xdr.LedgerEntryTypeTtl}

	ledgerChanges := map[xdr.LedgerEntryType]LedgerChanges{}
	ctx := context.Background()
	for seq := batchStart; seq <= batchEnd; {
		changeCompactors := map[xdr.LedgerEntryType]*ingest.ChangeCompactor{}
		for _, dt := range dataTypes {
			changeCompactors[dt] = ingest.NewChangeCompactor()
		}

		// if this ledger is available, we process its changes and move on to the next ledger by incrementing seq.
		// Otherwise, nothing is incremented, and we try again on the next iteration of the loop
		var header xdr.LedgerHeaderHistoryEntry
		if seq <= batchEnd {
			changeReader, err := ingest.NewLedgerChangeReader(ctx, *backend, "Public Global Stellar Network ; September 2015", seq)
			if err != nil {
				// this might need to be changed to continue if we are doing streaming
				return ChangeBatch{}
			}
			header = changeReader.LedgerTransactionReader.GetHeader()

			for {
				change, err := changeReader.Read()
				if err == io.EOF {
					break
				}
				if err != nil {
					return ChangeBatch{}
				}
				cache, ok := changeCompactors[change.Type]
				if !ok {
					// TODO: once LedgerEntryTypeData is tracked as well, all types should be addressed,
					// so this info log should be a warning.
					// Skip LedgerEntryTypeData as we are intentionally not processing it
					if change.Type != xdr.LedgerEntryTypeData {
					}
				} else {
					cache.AddChange(change)
				}
			}

			changeReader.Close()
			seq++
		}

		for dataType, compactor := range changeCompactors {
			for _, change := range compactor.GetChanges() {
				dataTypeChanges := ledgerChanges[dataType]
				dataTypeChanges.Changes = append(dataTypeChanges.Changes, change)
				dataTypeChanges.LedgerHeaders = append(dataTypeChanges.LedgerHeaders, header)
				ledgerChanges[dataType] = dataTypeChanges
			}
		}

	}

	return ChangeBatch{
		Changes:    ledgerChanges,
		BatchStart: batchStart,
		BatchEnd:   batchEnd,
	}
}
