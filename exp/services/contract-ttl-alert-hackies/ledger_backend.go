package main

import (
	"context"
	"time"

	"github.com/stellar/go/ingest/ledgerbackend"
	"github.com/stellar/go/support/datastore"
)

func CreateLedgerBackend(ctx context.Context) (ledgerbackend.LedgerBackend, error) {
	// Create ledger backend from captive core

	dataStore, err := CreateDatastore(ctx)
	if err != nil {
		return nil, err
	}

	BSBackendConfig := ledgerbackend.BufferedStorageBackendConfig{
		BufferSize: 200,
		NumWorkers: 10,
		RetryLimit: 3,
		RetryWait:  time.Duration(5) * time.Second,
	}

	backend, err := ledgerbackend.NewBufferedStorageBackend(BSBackendConfig, dataStore)
	if err != nil {
		return nil, err
	}
	return backend, nil
}

func CreateDatastore(ctx context.Context) (datastore.DataStore, error) {
	// These params are specific for GCS
	params := make(map[string]string)
	params["destination_bucket_path"] = "sdf-ledger-close-meta/ledgers/pubnet"
	dataStoreConfig := datastore.DataStoreConfig{
		Type:   "GCS",
		Params: params,
		// TODO: In the future these will come from a config file written by ledgerexporter
		// Hard code DataStoreSchema values for now
		Schema: datastore.DataStoreSchema{
			LedgersPerFile:    1,
			FilesPerPartition: 64000,
		},
	}

	return datastore.NewDataStore(ctx, dataStoreConfig)
}
