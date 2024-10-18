package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/big"
	"time"

	"github.com/dgryski/go-farm"
	"github.com/guregu/null"
	"github.com/stellar/go/amount"
	"github.com/stellar/go/ingest"
	"github.com/stellar/go/ingest/ledgerbackend"
	"github.com/stellar/go/toid"
	"github.com/stellar/go/xdr"
)

// OperationTransformInput is a representation of the input for the TransformOperation function
type OperationTransformInput struct {
	Operation       xdr.Operation
	OperationIndex  int32
	Transaction     ingest.LedgerTransaction
	LedgerSeqNum    int32
	LedgerCloseMeta xdr.LedgerCloseMeta
}

func panicIf(err error) {
	if err != nil {
		panic(fmt.Errorf("an error occurred, panicking: %s", err))
	}
}

// GetOperations returns a slice of operations for the ledgers in the provided range (inclusive on both ends)
func GetOperations(start, end uint32, limit int64) ([]OperationTransformInput, error) {
	ctx := context.Background()

	backend, err := CreateLedgerBackend(ctx)
	if err != nil {
		return []OperationTransformInput{}, err
	}

	opSlice := []OperationTransformInput{}
	err = backend.PrepareRange(ctx, ledgerbackend.BoundedRange(start, end))
	panicIf(err)
	for seq := start; seq <= end; seq++ {
		ledgerCloseMeta, err := backend.GetLedger(ctx, seq)
		if err != nil {
			return []OperationTransformInput{}, fmt.Errorf("error getting ledger seq %d from the backend: %v", seq, err)
		}

		txReader, err := ingest.NewLedgerTransactionReaderFromLedgerCloseMeta("Public Global Stellar Network ; September 2015", ledgerCloseMeta)
		if err != nil {
			return []OperationTransformInput{}, err
		}

		for int64(len(opSlice)) < limit || limit < 0 {
			tx, err := txReader.Read()
			if err == io.EOF {
				break
			}

			for index, op := range tx.Envelope.Operations() {
				opSlice = append(opSlice, OperationTransformInput{
					Operation:       op,
					OperationIndex:  int32(index),
					Transaction:     tx,
					LedgerSeqNum:    int32(seq),
					LedgerCloseMeta: ledgerCloseMeta,
				})

				if int64(len(opSlice)) >= limit && limit >= 0 {
					break
				}
			}
		}

		txReader.Close()

		if int64(len(opSlice)) >= limit && limit >= 0 {
			break
		}
	}

	return opSlice, nil
}

type OperationOutput struct {
	SourceAccount      string                 `json:"source_account"`
	SourceAccountMuxed string                 `json:"source_account_muxed,omitempty"`
	Type               int32                  `json:"type"`
	TypeString         string                 `json:"type_string"`
	OperationDetails   map[string]interface{} `json:"details"` //Details is a JSON object that varies based on operation type
	TransactionID      int64                  `json:"transaction_id"`
	OperationID        int64                  `json:"id"`
	ClosedAt           time.Time              `json:"closed_at"`
	LedgerSequence     uint32                 `json:"ledger_sequence"`
}

func TransformOperation(operation xdr.Operation, operationIndex int32, transaction ingest.LedgerTransaction, ledgerSeq int32, ledgerCloseMeta xdr.LedgerCloseMeta, network string) (OperationOutput, error) {
	outputTransactionID := toid.New(ledgerSeq, int32(transaction.Index), 0).ToInt64()
	outputOperationID := toid.New(ledgerSeq, int32(transaction.Index), operationIndex+1).ToInt64() //operationIndex needs +1 increment to stay in sync with ingest package

	sourceAccount := getOperationSourceAccount(operation, transaction)
	outputSourceAccount, err := GetAccountAddressFromMuxedAccount(sourceAccount)
	if err != nil {
		return OperationOutput{}, fmt.Errorf("for operation %d (ledger id=%d): %v", operationIndex, outputOperationID, err)
	}

	var outputSourceAccountMuxed null.String
	if sourceAccount.Type == xdr.CryptoKeyTypeKeyTypeMuxedEd25519 {
		muxedAddress, err := sourceAccount.GetAddress()
		if err != nil {
			return OperationOutput{}, err
		}
		outputSourceAccountMuxed = null.StringFrom(muxedAddress)
	}

	outputOperationType := int32(operation.Body.Type)
	if outputOperationType < 0 {
		return OperationOutput{}, fmt.Errorf("the operation type (%d) is negative for  operation %d (operation id=%d)", outputOperationType, operationIndex, outputOperationID)
	}

	outputDetails, err := extractOperationDetails(operation, transaction, operationIndex)
	if err != nil {
		return OperationOutput{}, err
	}

	outputOperationTypeString, err := mapOperationType(operation)
	if err != nil {
		return OperationOutput{}, err
	}

	outputCloseTime, err := GetCloseTime(ledgerCloseMeta)
	if err != nil {
		return OperationOutput{}, err
	}

	outputLedgerSequence := GetLedgerSequence(ledgerCloseMeta)

	transformedOperation := OperationOutput{
		SourceAccount:      outputSourceAccount,
		SourceAccountMuxed: outputSourceAccountMuxed.String,
		Type:               outputOperationType,
		TypeString:         outputOperationTypeString,
		TransactionID:      outputTransactionID,
		OperationID:        outputOperationID,
		OperationDetails:   outputDetails,
		ClosedAt:           outputCloseTime,
		LedgerSequence:     outputLedgerSequence,
	}

	return transformedOperation, nil
}

func getOperationSourceAccount(operation xdr.Operation, transaction ingest.LedgerTransaction) xdr.MuxedAccount {
	sourceAccount := operation.SourceAccount
	if sourceAccount != nil {
		return *sourceAccount
	}

	return transaction.Envelope.SourceAccount()
}

func GetAccountAddressFromMuxedAccount(account xdr.MuxedAccount) (string, error) {
	providedID := account.ToAccountId()
	pointerToID := &providedID
	return pointerToID.GetAddress()
}

func extractOperationDetails(operation xdr.Operation, transaction ingest.LedgerTransaction, operationIndex int32) (map[string]interface{}, error) {
	details := map[string]interface{}{}
	sourceAccount := getOperationSourceAccount(operation, transaction)
	operationType := operation.Body.Type

	switch operationType {
	case xdr.OperationTypeCreateAccount:
		op, ok := operation.Body.GetCreateAccountOp()
		if !ok {
			return details, fmt.Errorf("could not access CreateAccount info for this operation (index %d)", operationIndex)
		}

		if err := addAccountAndMuxedAccountDetails(details, sourceAccount, "funder"); err != nil {
			return details, err
		}
		details["account"] = op.Destination.Address()
		details["starting_balance"] = ConvertStroopValueToReal(op.StartingBalance)

	case xdr.OperationTypePayment:
		op, ok := operation.Body.GetPaymentOp()
		if !ok {
			return details, fmt.Errorf("could not access Payment info for this operation (index %d)", operationIndex)
		}

		if err := addAccountAndMuxedAccountDetails(details, sourceAccount, "from"); err != nil {
			return details, err
		}
		if err := addAccountAndMuxedAccountDetails(details, op.Destination, "to"); err != nil {
			return details, err
		}
		details["amount"] = ConvertStroopValueToReal(op.Amount)
		if err := addAssetDetailsToOperationDetails(details, op.Asset, ""); err != nil {
			return details, err
		}

	case xdr.OperationTypePathPaymentStrictReceive:
		op, ok := operation.Body.GetPathPaymentStrictReceiveOp()
		if !ok {
			return details, fmt.Errorf("could not access PathPaymentStrictReceive info for this operation (index %d)", operationIndex)
		}

		if err := addAccountAndMuxedAccountDetails(details, sourceAccount, "from"); err != nil {
			return details, err
		}
		if err := addAccountAndMuxedAccountDetails(details, op.Destination, "to"); err != nil {
			return details, err
		}
		details["amount"] = ConvertStroopValueToReal(op.DestAmount)
		details["source_amount"] = amount.String(0)
		details["source_max"] = ConvertStroopValueToReal(op.SendMax)
		if err := addAssetDetailsToOperationDetails(details, op.DestAsset, ""); err != nil {
			return details, err
		}
		if err := addAssetDetailsToOperationDetails(details, op.SendAsset, "source"); err != nil {
			return details, err
		}

		if transaction.Result.Successful() {
			allOperationResults, ok := transaction.Result.OperationResults()
			if !ok {
				return details, fmt.Errorf("could not access any results for this transaction")
			}
			currentOperationResult := allOperationResults[operationIndex]
			resultBody, ok := currentOperationResult.GetTr()
			if !ok {
				return details, fmt.Errorf("could not access result body for this operation (index %d)", operationIndex)
			}
			result, ok := resultBody.GetPathPaymentStrictReceiveResult()
			if !ok {
				return details, fmt.Errorf("could not access PathPaymentStrictReceive result info for this operation (index %d)", operationIndex)
			}
			details["source_amount"] = ConvertStroopValueToReal(result.SendAmount())
		}

		details["path"] = transformPath(op.Path)

	case xdr.OperationTypePathPaymentStrictSend:
		op, ok := operation.Body.GetPathPaymentStrictSendOp()
		if !ok {
			return details, fmt.Errorf("could not access PathPaymentStrictSend info for this operation (index %d)", operationIndex)
		}

		if err := addAccountAndMuxedAccountDetails(details, sourceAccount, "from"); err != nil {
			return details, err
		}
		if err := addAccountAndMuxedAccountDetails(details, op.Destination, "to"); err != nil {
			return details, err
		}
		details["amount"] = amount.String(0)
		details["source_amount"] = ConvertStroopValueToReal(op.SendAmount)
		details["destination_min"] = amount.String(op.DestMin)
		if err := addAssetDetailsToOperationDetails(details, op.DestAsset, ""); err != nil {
			return details, err
		}
		if err := addAssetDetailsToOperationDetails(details, op.SendAsset, "source"); err != nil {
			return details, err
		}

		if transaction.Result.Successful() {
			allOperationResults, ok := transaction.Result.OperationResults()
			if !ok {
				return details, fmt.Errorf("could not access any results for this transaction")
			}
			currentOperationResult := allOperationResults[operationIndex]
			resultBody, ok := currentOperationResult.GetTr()
			if !ok {
				return details, fmt.Errorf("could not access result body for this operation (index %d)", operationIndex)
			}
			result, ok := resultBody.GetPathPaymentStrictSendResult()
			if !ok {
				return details, fmt.Errorf("could not access GetPathPaymentStrictSendResult result info for this operation (index %d)", operationIndex)
			}
			details["amount"] = ConvertStroopValueToReal(result.DestAmount())
		}

		details["path"] = transformPath(op.Path)

	case xdr.OperationTypeAccountMerge:
		destinationAccount, ok := operation.Body.GetDestination()
		if !ok {
			return details, fmt.Errorf("could not access Destination info for this operation (index %d)", operationIndex)
		}

		if err := addAccountAndMuxedAccountDetails(details, sourceAccount, "account"); err != nil {
			return details, err
		}
		if err := addAccountAndMuxedAccountDetails(details, destinationAccount, "into"); err != nil {
			return details, err
		}
	}

	return details, nil
}

func mapOperationType(operation xdr.Operation) (string, error) {
	var op_string_type string
	operationType := operation.Body.Type

	switch operationType {
	case xdr.OperationTypeCreateAccount:
		op_string_type = "create_account"
	case xdr.OperationTypePayment:
		op_string_type = "payment"
	case xdr.OperationTypePathPaymentStrictReceive:
		op_string_type = "path_payment_strict_receive"
	case xdr.OperationTypePathPaymentStrictSend:
		op_string_type = "path_payment_strict_send"
	case xdr.OperationTypeManageBuyOffer:
		op_string_type = "manage_buy_offer"
	case xdr.OperationTypeManageSellOffer:
		op_string_type = "manage_sell_offer"
	case xdr.OperationTypeCreatePassiveSellOffer:
		op_string_type = "create_passive_sell_offer"
	case xdr.OperationTypeSetOptions:
		op_string_type = "set_options"
	case xdr.OperationTypeChangeTrust:
		op_string_type = "change_trust"
	case xdr.OperationTypeAllowTrust:
		op_string_type = "allow_trust"
	case xdr.OperationTypeAccountMerge:
		op_string_type = "account_merge"
	case xdr.OperationTypeInflation:
		op_string_type = "inflation"
	case xdr.OperationTypeManageData:
		op_string_type = "manage_data"
	case xdr.OperationTypeBumpSequence:
		op_string_type = "bump_sequence"
	case xdr.OperationTypeCreateClaimableBalance:
		op_string_type = "create_claimable_balance"
	case xdr.OperationTypeClaimClaimableBalance:
		op_string_type = "claim_claimable_balance"
	case xdr.OperationTypeBeginSponsoringFutureReserves:
		op_string_type = "begin_sponsoring_future_reserves"
	case xdr.OperationTypeEndSponsoringFutureReserves:
		op_string_type = "end_sponsoring_future_reserves"
	case xdr.OperationTypeRevokeSponsorship:
		op_string_type = "revoke_sponsorship"
	case xdr.OperationTypeClawback:
		op_string_type = "clawback"
	case xdr.OperationTypeClawbackClaimableBalance:
		op_string_type = "clawback_claimable_balance"
	case xdr.OperationTypeSetTrustLineFlags:
		op_string_type = "set_trust_line_flags"
	case xdr.OperationTypeLiquidityPoolDeposit:
		op_string_type = "liquidity_pool_deposit"
	case xdr.OperationTypeLiquidityPoolWithdraw:
		op_string_type = "liquidity_pool_withdraw"
	case xdr.OperationTypeInvokeHostFunction:
		op_string_type = "invoke_host_function"
	case xdr.OperationTypeExtendFootprintTtl:
		op_string_type = "extend_footprint_ttl"
	case xdr.OperationTypeRestoreFootprint:
		op_string_type = "restore_footprint"
	default:
		return op_string_type, fmt.Errorf("unknown operation type: %s", operation.Body.Type.String())
	}
	return op_string_type, nil
}

func GetCloseTime(lcm xdr.LedgerCloseMeta) (time.Time, error) {
	headerHistoryEntry := lcm.LedgerHeaderHistoryEntry()
	return ExtractLedgerCloseTime(headerHistoryEntry)
}

func ExtractLedgerCloseTime(ledger xdr.LedgerHeaderHistoryEntry) (time.Time, error) {
	return TimePointToUTCTimeStamp(ledger.Header.ScpValue.CloseTime)
}

func TimePointToUTCTimeStamp(providedTime xdr.TimePoint) (time.Time, error) {
	intTime := int64(providedTime)
	if intTime < 0 {
		return time.Now(), errors.New("the timepoint is negative")
	}
	return time.Unix(intTime, 0).UTC(), nil
}

func GetLedgerSequence(lcm xdr.LedgerCloseMeta) uint32 {
	headerHistoryEntry := lcm.LedgerHeaderHistoryEntry()
	return uint32(headerHistoryEntry.Header.LedgerSeq)
}

func addAccountAndMuxedAccountDetails(result map[string]interface{}, a xdr.MuxedAccount, prefix string) error {
	account_id := a.ToAccountId()
	result[prefix] = account_id.Address()
	prefix = formatPrefix(prefix)
	if a.Type == xdr.CryptoKeyTypeKeyTypeMuxedEd25519 {
		muxedAccountAddress, err := a.GetAddress()
		if err != nil {
			return err
		}
		result[prefix+"muxed"] = muxedAccountAddress
		muxedAccountId, err := a.GetId()
		if err != nil {
			return err
		}
		result[prefix+"muxed_id"] = muxedAccountId
	}
	return nil
}

func formatPrefix(p string) string {
	if p != "" {
		p += "_"
	}
	return p
}

func ConvertStroopValueToReal(input xdr.Int64) float64 {
	output, _ := big.NewRat(int64(input), int64(10000000)).Float64()
	return output
}

func addAssetDetailsToOperationDetails(result map[string]interface{}, asset xdr.Asset, prefix string) error {
	var assetType, code, issuer string
	err := asset.Extract(&assetType, &code, &issuer)
	if err != nil {
		return err
	}

	prefix = formatPrefix(prefix)
	result[prefix+"asset_type"] = assetType

	if asset.Type == xdr.AssetTypeAssetTypeNative {
		result[prefix+"asset_id"] = int64(-5706705804583548011)
		return nil
	}

	result[prefix+"asset_code"] = code
	result[prefix+"asset_issuer"] = issuer
	result[prefix+"asset_id"] = FarmHashAsset(code, issuer, assetType)

	return nil
}

func FarmHashAsset(assetCode, assetIssuer, assetType string) int64 {
	asset := fmt.Sprintf("%s%s%s", assetCode, assetIssuer, assetType)
	hash := farm.Fingerprint64([]byte(asset))

	return int64(hash)
}

type Path struct {
	AssetCode   string `json:"asset_code"`
	AssetIssuer string `json:"asset_issuer"`
	AssetType   string `json:"asset_type"`
}

func transformPath(initialPath []xdr.Asset) []Path {
	if len(initialPath) == 0 {
		return nil
	}
	var path = make([]Path, 0)
	for _, pathAsset := range initialPath {
		var assetType, code, issuer string
		err := pathAsset.Extract(&assetType, &code, &issuer)
		if err != nil {
			return nil
		}

		path = append(path, Path{
			AssetType:   assetType,
			AssetIssuer: issuer,
			AssetCode:   code,
		})
	}
	return path
}
