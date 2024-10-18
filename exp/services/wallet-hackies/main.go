package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	_ "github.com/marcboeker/go-duckdb"
)

func main() {
	// latest sequence in duckdb
	// inital parquet files at 53979576; < 2024-10-17
	//sequence := GetLatestSequence()

	// watchlist of contract_ids to alert ttl on
	file, err := os.Open("watchlist.json")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	// Read the file's contents
	byteValue, _ := ioutil.ReadAll(file)

	// Unmarshal the JSON into a map
	var results []map[string]interface{}
	err = json.Unmarshal(byteValue, &results)
	if err != nil {
		log.Fatal(err)
	}

	var accounts []string
	for _, result := range results {
		accounts = append(accounts, result["account"].(string))
	}

	ledgers := GetPayments(accounts)
	//fmt.Printf("num ledgers: %d\n", len(ledgers))

	for _, ledger := range ledgers {
		operations, _ := GetOperations(uint32(ledger), uint32(ledger), -1)
		for _, transformInput := range operations {
			transformed, _ := TransformOperation(transformInput.Operation, transformInput.OperationIndex, transformInput.Transaction, transformInput.LedgerSeqNum, transformInput.LedgerCloseMeta, "Public Global Stellar Network ; September 2015")

			inserted := false
			_, exists := transformed.OperationDetails["account"]
			if exists {
				//fmt.Printf("account: %s\n", transformed.OperationDetails["account"])
				for _, account := range accounts {
					if transformed.OperationDetails["account"] == account {
						InsertPayment(transformed)
						inserted = true
					}
				}
				if inserted {
					continue
				}
			}

			_, exists = transformed.OperationDetails["from"]
			if exists {
				//fmt.Printf("account: %s\n", transformed.OperationDetails["from"])
				for _, account := range accounts {
					if transformed.OperationDetails["from"] == account {
						InsertPayment(transformed)
						inserted = true
					}
				}
				if inserted {
					continue
				}
			}

			_, exists = transformed.OperationDetails["to"]
			if exists {
				//fmt.Printf("account: %s\n", transformed.OperationDetails["to"])
				for _, account := range accounts {
					if transformed.OperationDetails["to"] == account {
						InsertPayment(transformed)
						inserted = true
					}
				}
				if inserted {
					continue
				}
			}

			_, exists = transformed.OperationDetails["into"]
			if exists {
				//fmt.Printf("account: %s\n", transformed.OperationDetails["into"])
				for _, account := range accounts {
					if transformed.OperationDetails["into"] == account {
						InsertPayment(transformed)
						inserted = true
					}
				}
				if inserted {
					continue
				}
			}

			_, exists = transformed.OperationDetails["source_account"]
			if exists {
				//fmt.Printf("account: %s\n", transformed.OperationDetails["source_account"])
				for _, account := range accounts {
					if transformed.OperationDetails["source_account"] == account {
						InsertPayment(transformed)
						inserted = true
					}
				}
				if inserted {
					continue
				}
			}

		}
	}

	//ctx := context.Background()

	//backend, _ := CreateLedgerBackend(ctx)

	//start := uint32(sequence)
	//backend.PrepareRange(ctx, ledgerbackend.UnboundedRange(start))
	//LedgerStreamChanges(&backend, start, addresses)

	//seq := uint32(53973547)
	//for {
	//	lcm, _ := backend.GetLedger(ctx, seq)

	//	header := lcm.LedgerHeaderHistoryEntry()
	//	out := header.Header.LedgerSeq
	//	fmt.Printf("ledger: %d\n", uint32(out))

	//	seq += 1
	//}

	//db, _ := sql.Open("duckdb", "index_database.duckdb")
	//defer db.Close()

	//var sequence int64
	//query := `SELECT sequence FROM latest_ledger_sequence LIMIT 1;`

	//db.QueryRow(query).Scan(&sequence)

	//db.Exec(`DELETE FROM latest_ledger_sequence`)
	//db.Exec(`INSERT INTO latest_ledger_sequence (sequence) VALUES (?)`, sequence+1)
}

func InsertPayment(op OperationOutput) {
	db, err := sql.Open("duckdb", "payment_database.duckdb")
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}
	defer db.Close()

	detailsJSON, err := json.Marshal(op.OperationDetails)
	if err != nil {
		log.Fatal("Failed to marshal operation details:", err)
	}

	query := `
		INSERT INTO payment_operations (
			source_account, source_account_muxed, type, type_string, operation_details, 
			transaction_id, operation_id, closed_at, ledger_sequence
		)
		SELECT ?, ?, ?, ?, ?, ?, ?, ?, ?
		WHERE NOT EXISTS (
			SELECT 1 FROM payment_operations WHERE operation_id = ?
		);
	`

	_, err = db.Exec(query,
		op.SourceAccount,
		op.SourceAccountMuxed,
		op.Type,
		op.TypeString,
		string(detailsJSON),
		op.TransactionID,
		op.OperationID,
		op.ClosedAt,
		op.LedgerSequence,
		op.OperationID,
	)
	if err != nil {
		log.Fatal("Failed to insert data:", err)
	}

	fmt.Printf("ledger: %d; op_id: %d\n", op.LedgerSequence, op.OperationID)
}
