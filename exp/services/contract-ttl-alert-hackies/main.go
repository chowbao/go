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
	sequence := GetLatestSequence()

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

	var contractAlerts []string
	for _, result := range results {
		contractAlerts = append(contractAlerts, result["contract_id"].(string))
	}

	GetKeyHashAlerts(contractAlerts)

	ctx := context.Background()

	backend, _ := CreateLedgerBackend(ctx)

	start := uint32(sequence)
	backend.PrepareRange(ctx, ledgerbackend.UnboundedRange(start))
	LedgerStreamChanges(&backend, start, contractAlerts)

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
