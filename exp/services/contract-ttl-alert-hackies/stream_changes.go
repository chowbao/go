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

func LedgerStreamChanges(backend *ledgerbackend.LedgerBackend, start uint32, contractAlerts []string) {
	changeChan := make(chan ChangeBatch)
	closeChan := make(chan int)
	alerted := make(map[string]bool)

	go StreamChanges(backend, start, 93973547, 1, changeChan, closeChan)

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

						db.Exec(`INSERT INTO contract_data (contract_id, ledger_key_hash, ledger_sequence) VALUES (?,?,?)`, contractData.ContractId, contractData.LedgerKeyHash, contractData.LedgerSequence)

						db.Exec(`DELETE FROM latest_ledger_sequence`)
						db.Exec(`INSERT INTO latest_ledger_sequence (sequence) VALUES (?)`, contractData.LedgerSequence)

						//fmt.Printf("ledger_sequence: %d\n", contractData.LedgerSequence)
					}
				case xdr.LedgerEntryTypeContractCode:
					for i, change := range changes.Changes {
						contractCode := TransformContractCode(change, changes.LedgerHeaders[i])
						transformedOutputs["contract_code"] = append(transformedOutputs["contract_code"], contractCode)

						db, _ := sql.Open("duckdb", "index_database.duckdb")
						defer db.Close()

						db.Exec(`INSERT INTO contract_code (contract_code_hash, ledger_key_hash, ledger_sequence) VALUES (?,?,?)`, contractCode.ContractCodeHash, contractCode.LedgerKeyHash, contractCode.LedgerSequence)

						db.Exec(`DELETE FROM latest_ledger_sequence`)
						db.Exec(`INSERT INTO latest_ledger_sequence (sequence) VALUES (?)`, contractCode.LedgerSequence)

						//fmt.Printf("ledger_sequence: %d\n", contractCode.LedgerSequence)
					}
				case xdr.LedgerEntryTypeTtl:
					for i, change := range changes.Changes {
						ttl := TransformTtl(change, changes.LedgerHeaders[i])
						transformedOutputs["ttl"] = append(transformedOutputs["ttl"], ttl)

						db, _ := sql.Open("duckdb", "index_database.duckdb")
						defer db.Close()

						db.Exec(`INSERT INTO ttl (key_hash, live_until_ledger_seq, ledger_sequence) VALUES (?,?,?)`, ttl.KeyHash, ttl.LiveUntilLedgerSeq, ttl.LedgerSequence)

						db.Exec(`DELETE FROM latest_ledger_sequence`)
						db.Exec(`INSERT INTO latest_ledger_sequence (sequence) VALUES (?)`, ttl.LedgerSequence)

						//fmt.Printf("ledger_sequence: %d\n", ttl.LedgerSequence)
					}
				default:
					ledgerSequence := uint32(changes.LedgerHeaders[0].Header.LedgerSeq)
					db, _ := sql.Open("duckdb", "index_database.duckdb")
					defer db.Close()

					db.Exec(`DELETE FROM latest_ledger_sequence`)
					db.Exec(`INSERT INTO latest_ledger_sequence (sequence) VALUES (?)`, ledgerSequence)

					//fmt.Printf("ledger_sequence: %d\n", ledgerSequence)
				}
			}

			// check if contract ttl expired
			GetKeyHashAlerts(contractAlerts)
			sequence := GetLatestSequence()

			db, _ := sql.Open("duckdb", "index_database.duckdb")
			defer db.Close()

			for _, contract := range contractAlerts {
				var ttl int64
				query := `select ttl from contract_ttl_alerts where contract_id = ?;`
				err := db.QueryRow(query, contract).Scan(&ttl)
				if err != nil {
					continue
				}
				_, exists := alerted[contract]
				if !exists {
					alerted[contract] = false
				}

				if ttl < sequence {
					if !alerted[contract] {
						fmt.Printf("contract %s expired at %d\n", contract, ttl)
						alerted[contract] = true
					}
				} else {
					alerted[contract] = false
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
