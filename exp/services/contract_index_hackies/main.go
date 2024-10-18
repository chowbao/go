package main

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"os"

	_ "github.com/marcboeker/go-duckdb"
	"github.com/stellar/go/ingest/ledgerbackend"
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

	var contracts []string
	for _, result := range results {
		contracts = append(contracts, result["contract_id"].(string))
	}

	ledgers := GetContractData(contracts)
	//fmt.Printf("num ledgers: %d\n", len(ledgers))

	for _, ledger := range ledgers {
		ctx := context.Background()
		backend, _ := CreateLedgerBackend(ctx)
		backend.PrepareRange(ctx, ledgerbackend.BoundedRange(uint32(ledger), uint32(ledger)+1))

		LedgerStreamChanges(&backend, uint32(ledger), uint32(ledger)+1, contracts)
	}
}
