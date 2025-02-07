package utils

import (
	"time"

	"github.com/guregu/null"
	"github.com/lib/pq"
	"github.com/stellar/go/historyarchive"
	"github.com/stellar/go/xdr"
)

// TransactionOutput is a representation of a transaction that aligns with the BigQuery table history_transactions
type TransactionOutput struct {
	TransactionHash                      string         `json:"transaction_hash"`
	LedgerSequence                       uint32         `json:"ledger_sequence"`
	Account                              string         `json:"account"`
	AccountMuxed                         string         `json:"account_muxed,omitempty"`
	AccountSequence                      int64          `json:"account_sequence"`
	MaxFee                               uint32         `json:"max_fee"`
	FeeCharged                           int64          `json:"fee_charged"`
	OperationCount                       int32          `json:"operation_count"`
	TxEnvelope                           string         `json:"tx_envelope"`
	TxResult                             string         `json:"tx_result"`
	TxMeta                               string         `json:"tx_meta"`
	TxFeeMeta                            string         `json:"tx_fee_meta"`
	CreatedAt                            time.Time      `json:"created_at"`
	MemoType                             string         `json:"memo_type"`
	Memo                                 string         `json:"memo"`
	TimeBounds                           string         `json:"time_bounds"`
	Successful                           bool           `json:"successful"`
	TransactionID                        int64          `json:"id"`
	FeeAccount                           string         `json:"fee_account,omitempty"`
	FeeAccountMuxed                      string         `json:"fee_account_muxed,omitempty"`
	InnerTransactionHash                 string         `json:"inner_transaction_hash,omitempty"`
	NewMaxFee                            uint32         `json:"new_max_fee,omitempty"`
	LedgerBounds                         string         `json:"ledger_bounds"`
	MinAccountSequence                   null.Int       `json:"min_account_sequence"`
	MinAccountSequenceAge                null.Int       `json:"min_account_sequence_age"`
	MinAccountSequenceLedgerGap          null.Int       `json:"min_account_sequence_ledger_gap"`
	ExtraSigners                         pq.StringArray `json:"extra_signers"`
	ClosedAt                             time.Time      `json:"closed_at"`
	ResourceFee                          int64          `json:"resource_fee"`
	SorobanResourcesInstructions         uint32         `json:"soroban_resources_instructions"`
	SorobanResourcesReadBytes            uint32         `json:"soroban_resources_read_bytes"`
	SorobanResourcesWriteBytes           uint32         `json:"soroban_resources_write_bytes"`
	TransactionResultCode                string         `json:"transaction_result_code"`
	InclusionFeeBid                      int64          `json:"inclusion_fee_bid"`
	InclusionFeeCharged                  int64          `json:"inclusion_fee_charged"`
	ResourceFeeRefund                    int64          `json:"resource_fee_refund"`
	TotalNonRefundableResourceFeeCharged int64          `json:"non_refundable_resource_fee_charged"`
	TotalRefundableResourceFeeCharged    int64          `json:"refundable_resource_fee_charged"`
	RentFeeCharged                       int64          `json:"rent_fee_charged"`
	TxSigners                            []string       `json:"tx_signers"`
}

type LedgerTransactionOutput struct {
	LedgerSequence  uint32    `json:"ledger_sequence"`
	TxEnvelope      string    `json:"tx_envelope"`
	TxResult        string    `json:"tx_result"`
	TxMeta          string    `json:"tx_meta"`
	TxFeeMeta       string    `json:"tx_fee_meta"`
	TxLedgerHistory string    `json:"tx_ledger_history"`
	ClosedAt        time.Time `json:"closed_at"`
}

// OperationOutput is a representation of an operation that aligns with the BigQuery table history_operations
type OperationOutput struct {
	SourceAccount        string                 `json:"source_account"`
	SourceAccountMuxed   string                 `json:"source_account_muxed,omitempty"`
	Type                 int32                  `json:"type"`
	TypeString           string                 `json:"type_string"`
	OperationDetails     map[string]interface{} `json:"details"` //Details is a JSON object that varies based on operation type
	TransactionID        int64                  `json:"transaction_id"`
	OperationID          int64                  `json:"id"`
	ClosedAt             time.Time              `json:"closed_at"`
	OperationResultCode  string                 `json:"operation_result_code"`
	OperationTraceCode   string                 `json:"operation_trace_code"`
	LedgerSequence       uint32                 `json:"ledger_sequence"`
	OperationDetailsJSON map[string]interface{} `json:"details_json"`
}

// Claimants
type Claimant struct {
	Destination string             `json:"destination"`
	Predicate   xdr.ClaimPredicate `json:"predicate"`
}

// Price represents the price of an asset as a fraction
type Price struct {
	Numerator   int32 `json:"n"`
	Denominator int32 `json:"d"`
}

// Path is a representation of an asset without an ID that forms part of a path in a path payment
type Path struct {
	AssetCode   string `json:"asset_code"`
	AssetIssuer string `json:"asset_issuer"`
	AssetType   string `json:"asset_type"`
}

// LiquidityPoolAsset represents the asset pairs in a liquidity pool
type LiquidityPoolAsset struct {
	AssetAType   string
	AssetACode   string
	AssetAIssuer string
	AssetAAmount float64
	AssetBType   string
	AssetBCode   string
	AssetBIssuer string
	AssetBAmount float64
}

type SponsorshipOutput struct {
	Operation      xdr.Operation
	OperationIndex uint32
}

// TradeEffectDetails is a struct of data from `effects.DetailsString`
// when the effect type is trade
type TradeEffectDetails struct {
	Seller            string `json:"seller"`
	SellerMuxed       string `json:"seller_muxed,omitempty"`
	SellerMuxedID     uint64 `json:"seller_muxed_id,omitempty"`
	OfferID           int64  `json:"offer_id"`
	SoldAmount        string `json:"sold_amount"`
	SoldAssetType     string `json:"sold_asset_type"`
	SoldAssetCode     string `json:"sold_asset_code,omitempty"`
	SoldAssetIssuer   string `json:"sold_asset_issuer,omitempty"`
	BoughtAmount      string `json:"bought_amount"`
	BoughtAssetType   string `json:"bought_asset_type"`
	BoughtAssetCode   string `json:"bought_asset_code,omitempty"`
	BoughtAssetIssuer string `json:"bought_asset_issuer,omitempty"`
}

// TestTransaction transaction meta
type TestTransaction struct {
	Index         uint32
	EnvelopeXDR   string
	ResultXDR     string
	FeeChangesXDR string
	MetaXDR       string
	Hash          string
}

// ContractEventOutput is a representation of soroban contract events and diagnostic events
type ContractEventOutput struct {
	TransactionHash          string                         `json:"transaction_hash"`
	TransactionID            int64                          `json:"transaction_id"`
	Successful               bool                           `json:"successful"`
	LedgerSequence           uint32                         `json:"ledger_sequence"`
	ClosedAt                 time.Time                      `json:"closed_at"`
	InSuccessfulContractCall bool                           `json:"in_successful_contract_call"`
	ContractId               string                         `json:"contract_id"`
	Type                     int32                          `json:"type"`
	TypeString               string                         `json:"type_string"`
	Topics                   map[string][]map[string]string `json:"topics"`
	TopicsDecoded            map[string][]map[string]string `json:"topics_decoded"`
	Data                     map[string]string              `json:"data"`
	DataDecoded              map[string]string              `json:"data_decoded"`
	ContractEventXDR         string                         `json:"contract_event_xdr"`
}

type HistoryArchiveLedgerAndLCM struct {
	Ledger historyarchive.Ledger
	LCM    xdr.LedgerCloseMeta
}
