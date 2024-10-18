package main

import (
	"database/sql"
	"fmt"
	"log"
)

func GetLatestSequence() int64 {
	db, _ := sql.Open("duckdb", "index_database.duckdb")
	defer db.Close()

	var sequence int64
	query := `SELECT sequence FROM latest_ledger_sequence LIMIT 1;`

	db.QueryRow(query).Scan(&sequence)

	return sequence
}

func GetContractData(contracts []string) []int {
	db, _ := sql.Open("duckdb", "index_database.duckdb")
	defer db.Close()

	var contractString string
	for _, contract := range contracts {
		contractString = contractString + `'` + contract + `',`
	}

	query := fmt.Sprintf(`SELECT ledger_sequence FROM contract_data where contract_id in (%s)`, contractString[:len(contractString)-1])

	// Execute the query
	rows, err := db.Query(query)
	//rows, err := db.Query(query, accountString[:len(accountString)-1])
	if err != nil {
		log.Fatal("Failed to execute query:", err)
	}
	defer rows.Close()

	var ledgerSequences []int

	for rows.Next() {
		var ledgerSequence int
		if err := rows.Scan(&ledgerSequence); err != nil {
			log.Fatal("Failed to scan row:", err)
		}
		ledgerSequences = append(ledgerSequences, ledgerSequence)
	}
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}

	return ledgerSequences
}
