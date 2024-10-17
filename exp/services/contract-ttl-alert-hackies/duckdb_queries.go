package main

import (
	"database/sql"
	"log"
	"strconv"
)

func GetLatestSequence() int64 {
	db, _ := sql.Open("duckdb", "index_database.duckdb")
	defer db.Close()

	var sequence int64
	query := `SELECT sequence FROM latest_ledger_sequence LIMIT 1;`

	db.QueryRow(query).Scan(&sequence)

	return sequence
}

func AddContractAlert(contract, keyHash string, ttl uint32) {
	db, _ := sql.Open("duckdb", "index_database.duckdb")
	defer db.Close()

	queryCheck := `SELECT EXISTS(SELECT 1 FROM contract_ttl_alerts WHERE key_hash = ?);`

	var exists bool
	err := db.QueryRow(queryCheck, keyHash).Scan(&exists)
	if err != nil {
		log.Fatal(err)
	}

	if exists {
		// If it exists, update the ttl
		queryUpdate := `UPDATE contract_ttl_alerts SET ttl = ? WHERE key_hash = ?;`
		_, err = db.Exec(queryUpdate, ttl, keyHash)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		// If it doesn't exist, insert the new row
		queryInsert := `INSERT INTO contract_ttl_alerts (contract_id, key_hash, ttl) VALUES (?, ?, ?);`
		_, err = db.Exec(queryInsert, contract, keyHash, ttl)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func GetKeyHashAlerts(contractAlerts []string) {
	// get the ledger_key_hash and ttl
	for _, contract := range contractAlerts {
		db, _ := sql.Open("duckdb", "index_database.duckdb")
		defer db.Close()

		var keyHash string
		query := `select ledger_key_hash from contract_data where contract_key_type = 'ScValTypeScvLedgerKeyContractInstance' and contract_id = ? order by ledger_sequence asc limit 1;`

		err := db.QueryRow(query, contract).Scan(&keyHash)
		if err != nil {
			log.Fatal(err)
		}

		var ttl string
		query = `select max(live_until_ledger_seq) as live_until_ledger_seq from ttl where key_hash = ?;`
		db.QueryRow(query, keyHash).Scan(&ttl)

		uint_ttl, _ := strconv.ParseUint(ttl, 10, 64)

		AddContractAlert(contract, keyHash, uint32(uint_ttl))
	}
}
