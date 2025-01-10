package ingest

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strconv"

	"github.com/stellar/go/strkey"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/toid"
	"github.com/stellar/go/xdr"
)

// LedgerTransaction represents the data for a single transaction within a ledger.
type LedgerTransaction struct {
	Index    uint32
	Envelope xdr.TransactionEnvelope
	Result   xdr.TransactionResultPair
	// FeeChanges and UnsafeMeta are low level values, do not use them directly unless
	// you know what you are doing.
	// Use LedgerTransaction.GetChanges() for higher level access to ledger
	// entry changes.
	FeeChanges    xdr.LedgerEntryChanges
	UnsafeMeta    xdr.TransactionMeta
	LedgerVersion uint32
	Ledger        xdr.LedgerCloseMeta // This is read-only and not to be modified by downstream functions
	Hash          xdr.Hash
}

func (t *LedgerTransaction) txInternalError() bool {
	return t.Result.Result.Result.Code == xdr.TransactionResultCodeTxInternalError
}

// GetFeeChanges returns a developer friendly representation of LedgerEntryChanges
// connected to fees.
func (t *LedgerTransaction) GetFeeChanges() []Change {
	changes := GetChangesFromLedgerEntryChanges(t.FeeChanges)
	for i := range changes {
		changes[i].Reason = LedgerEntryChangeReasonFee
		changes[i].Transaction = t
	}
	return changes
}

func (t *LedgerTransaction) getTransactionChanges(ledgerEntryChanges xdr.LedgerEntryChanges) []Change {
	changes := GetChangesFromLedgerEntryChanges(ledgerEntryChanges)
	for i := range changes {
		changes[i].Reason = LedgerEntryChangeReasonTransaction
		changes[i].Transaction = t
	}
	return changes
}

// GetChanges returns a developer friendly representation of LedgerEntryChanges.
// It contains transaction changes and operation changes in that order. If the
// transaction failed with TxInternalError, operations and txChangesAfter are
// omitted. It doesn't support legacy TransactionMeta.V=0.
func (t *LedgerTransaction) GetChanges() ([]Change, error) {
	var changes []Change

	// Transaction meta
	switch t.UnsafeMeta.V {
	case 0:
		return changes, errors.New("TransactionMeta.V=0 not supported")
	case 1:
		v1Meta := t.UnsafeMeta.MustV1()
		// The var `txChanges` reflect the ledgerEntryChanges that are changed because of the transaction as a whole
		txChanges := t.getTransactionChanges(v1Meta.TxChanges)
		changes = append(changes, txChanges...)

		// Ignore operations meta if txInternalError https://github.com/stellar/go/issues/2111
		if t.txInternalError() && t.LedgerVersion <= 12 {
			return changes, nil
		}

		// These changes reflect the ledgerEntry changes that were caused by the operations in the transaction
		// Populate the operationInfo for these changes in the `Change` struct

		operationMeta := v1Meta.Operations
		//	operationMeta is a list of lists.
		//	Each element in operationMeta is a list of ledgerEntryChanges
		//	caused by the operation at that index of the element
		for opIdx := range operationMeta {
			opChanges := t.operationChanges(v1Meta.Operations, uint32(opIdx))
			changes = append(changes, opChanges...)
		}
	case 2, 3:
		var (
			txBeforeChanges, txAfterChanges xdr.LedgerEntryChanges
			operationMeta                   []xdr.OperationMeta
		)

		switch t.UnsafeMeta.V {
		case 2:
			v2Meta := t.UnsafeMeta.MustV2()
			txBeforeChanges = v2Meta.TxChangesBefore
			txAfterChanges = v2Meta.TxChangesAfter
			operationMeta = v2Meta.Operations
		case 3:
			v3Meta := t.UnsafeMeta.MustV3()
			txBeforeChanges = v3Meta.TxChangesBefore
			txAfterChanges = v3Meta.TxChangesAfter
			operationMeta = v3Meta.Operations
		default:
			panic("Invalid meta version, expected 2 or 3")
		}

		txChangesBefore := t.getTransactionChanges(txBeforeChanges)
		changes = append(changes, txChangesBefore...)

		// Ignore operations meta and txChangesAfter if txInternalError
		// https://github.com/stellar/go/issues/2111
		if t.txInternalError() && t.LedgerVersion <= 12 {
			return changes, nil
		}

		//	operationMeta is a list of lists.
		//	Each element in operationMeta is a list of ledgerEntryChanges
		//	caused by the operation at that index of the element
		for opIdx := range operationMeta {
			opChanges := t.operationChanges(operationMeta, uint32(opIdx))
			changes = append(changes, opChanges...)
		}

		txChangesAfter := t.getTransactionChanges(txAfterChanges)
		changes = append(changes, txChangesAfter...)
	default:
		return changes, errors.New("Unsupported TransactionMeta version")
	}

	return changes, nil
}

// GetOperation returns an operation by index.
func (t *LedgerTransaction) GetOperation(index uint32) (xdr.Operation, bool) {
	ops := t.Envelope.Operations()
	if int(index) >= len(ops) {
		return xdr.Operation{}, false
	}
	return ops[index], true
}

// GetOperationChanges returns a developer friendly representation of LedgerEntryChanges.
// It contains only operation changes.
func (t *LedgerTransaction) GetOperationChanges(operationIndex uint32) ([]Change, error) {
	if t.UnsafeMeta.V == 0 {
		return []Change{}, errors.New("TransactionMeta.V=0 not supported")
	}

	// Ignore operations meta if txInternalError https://github.com/stellar/go/issues/2111
	if t.txInternalError() && t.LedgerVersion <= 12 {
		return []Change{}, nil
	}

	var operationMeta []xdr.OperationMeta
	switch t.UnsafeMeta.V {
	case 1:
		operationMeta = t.UnsafeMeta.MustV1().Operations
	case 2:
		operationMeta = t.UnsafeMeta.MustV2().Operations
	case 3:
		operationMeta = t.UnsafeMeta.MustV3().Operations
	default:
		return []Change{}, errors.New("Unsupported TransactionMeta version")
	}

	return t.operationChanges(operationMeta, operationIndex), nil
}

func (t *LedgerTransaction) operationChanges(ops []xdr.OperationMeta, index uint32) []Change {
	if int(index) >= len(ops) {
		return []Change{}
	}

	operationMeta := ops[index]
	changes := GetChangesFromLedgerEntryChanges(operationMeta.Changes)

	for i := range changes {
		changes[i].Reason = LedgerEntryChangeReasonOperation
		changes[i].Transaction = t
		changes[i].OperationIndex = index
	}
	return changes
}

// GetDiagnosticEvents returns all contract events emitted by a given operation.
func (t *LedgerTransaction) GetDiagnosticEvents() ([]xdr.DiagnosticEvent, error) {
	return t.UnsafeMeta.GetDiagnosticEvents()
}

func (t *LedgerTransaction) ID() int64 {
	return toid.New(int32(t.Ledger.LedgerSequence()), int32(t.Index), 0).ToInt64()
}

func (t *LedgerTransaction) Account() (string, error) {
	sourceAccount := t.Envelope.SourceAccount()
	providedID := sourceAccount.ToAccountId()
	pointerToID := &providedID

	return pointerToID.GetAddress()
}

func (t *LedgerTransaction) AccountSequence() int64 {
	return t.Envelope.SeqNum()
}

func (t *LedgerTransaction) MaxFee() uint32 {
	return t.Envelope.Fee()
}

func (t *LedgerTransaction) FeeCharged() (int64, bool) {
	// Any Soroban Fee Bump transactions before P21 will need the below logic to calculate the correct feeCharged
	// Protocol 20 contained a bug where the feeCharged was incorrectly calculated but was fixed for
	// Protocol 21 with https://github.com/stellar/stellar-core/issues/4188
	var ok bool
	_, ok = t.GetSorobanData()
	if ok {
		if uint32(t.Ledger.LedgerHeaderHistoryEntry().Header.LedgerVersion) < 21 && t.Envelope.Type == xdr.EnvelopeTypeEnvelopeTypeTxFeeBump {
			var resourceFeeRefund int64
			var inclusionFeeCharged int64

			resourceFeeRefund, ok = t.SorobanResourceFeeRefund()
			if !ok {
				return 0, false
			}

			inclusionFeeCharged, ok = t.SorobanInclusionFeeCharged()
			if !ok {
				return 0, false
			}

			return int64(t.Result.Result.FeeCharged) - resourceFeeRefund + inclusionFeeCharged, true
		}
	}

	return int64(t.Result.Result.FeeCharged), true

}

func (t *LedgerTransaction) OperationCount() uint32 {
	return uint32(len(t.Envelope.Operations()))
}

func (t *LedgerTransaction) Memo() string {
	memoObject := t.Envelope.Memo()
	memoContents := ""
	switch xdr.MemoType(memoObject.Type) {
	case xdr.MemoTypeMemoText:
		memoContents = memoObject.MustText()
	case xdr.MemoTypeMemoId:
		memoContents = strconv.FormatUint(uint64(memoObject.MustId()), 10)
	case xdr.MemoTypeMemoHash:
		hash := memoObject.MustHash()
		memoContents = base64.StdEncoding.EncodeToString(hash[:])
	case xdr.MemoTypeMemoReturn:
		hash := memoObject.MustRetHash()
		memoContents = base64.StdEncoding.EncodeToString(hash[:])
	}

	return memoContents
}

func (t *LedgerTransaction) MemoType() string {
	memoObject := t.Envelope.Memo()
	return memoObject.Type.String()
}

func (t *LedgerTransaction) TimeBounds() (string, error) {
	timeBounds := t.Envelope.TimeBounds()
	if timeBounds == nil {
		return "", nil
	}

	if timeBounds.MaxTime < timeBounds.MinTime && timeBounds.MaxTime != 0 {
		return "", fmt.Errorf("the max time is earlier than the min time")
	}

	if timeBounds.MaxTime == 0 {
		return fmt.Sprintf("[%d,)", timeBounds.MinTime), nil
	}

	return fmt.Sprintf("[%d,%d)", timeBounds.MinTime, timeBounds.MaxTime), nil
}

func (t *LedgerTransaction) LedgerBounds() (string, bool) {
	ledgerBounds := t.Envelope.LedgerBounds()
	if ledgerBounds == nil {
		return "", false
	}

	return fmt.Sprintf("[%d,%d)", int64(ledgerBounds.MinLedger), int64(ledgerBounds.MaxLedger)), true

}

func (t *LedgerTransaction) MinSequence() (int64, bool) {
	minSeqNum := t.Envelope.MinSeqNum()
	if minSeqNum == nil {
		return 0, false
	}

	return *minSeqNum, true
}

func (t *LedgerTransaction) MinSequenceAge() (int64, bool) {
	minSequenceAge := t.Envelope.MinSeqAge()
	if minSequenceAge == nil {
		return 0, false
	}

	return int64(*minSequenceAge), true
}

func (t *LedgerTransaction) MinSequenceLedgerGap() (int64, bool) {
	minSequenceLedgerGap := t.Envelope.MinSeqLedgerGap()
	if minSequenceLedgerGap == nil {
		return 0, false
	}

	return int64(*minSequenceLedgerGap), true
}

func (t *LedgerTransaction) GetSorobanData() (xdr.SorobanTransactionData, bool) {
	switch t.Envelope.Type {
	case xdr.EnvelopeTypeEnvelopeTypeTx:
		return t.Envelope.V1.Tx.Ext.GetSorobanData()
	case xdr.EnvelopeTypeEnvelopeTypeTxFeeBump:
		return t.Envelope.FeeBump.Tx.InnerTx.V1.Tx.Ext.GetSorobanData()
	default:
		return xdr.SorobanTransactionData{}, false
	}
}

func (t *LedgerTransaction) SorobanResourceFee() (int64, bool) {
	sorobanData, ok := t.GetSorobanData()
	if !ok {
		return 0, false
	}

	return int64(sorobanData.ResourceFee), true
}

func (t *LedgerTransaction) SorobanResourcesInstructions() (uint32, bool) {
	sorobanData, ok := t.GetSorobanData()
	if !ok {
		return 0, false
	}

	return uint32(sorobanData.Resources.Instructions), true
}

func (t *LedgerTransaction) SorobanResourcesReadBytes() (uint32, bool) {
	sorobanData, ok := t.GetSorobanData()
	if !ok {
		return 0, false
	}

	return uint32(sorobanData.Resources.ReadBytes), true
}

func (t *LedgerTransaction) SorobanResourcesWriteBytes() (uint32, bool) {
	sorobanData, ok := t.GetSorobanData()
	if !ok {
		return 0, false
	}

	return uint32(sorobanData.Resources.WriteBytes), true
}

func (t *LedgerTransaction) InclusionFeeBid() (int64, bool) {
	resourceFee, ok := t.SorobanResourceFee()
	if !ok {
		return 0, false
	}

	return int64(t.Envelope.Fee()) - resourceFee, true
}

func (t *LedgerTransaction) FeeAccountAddress() (string, bool) {
	switch t.Envelope.Type {
	case xdr.EnvelopeTypeEnvelopeTypeTx:
		sourceAccount := t.Envelope.SourceAccount()
		feeAccountAddress := sourceAccount.Address()
		return feeAccountAddress, true
	case xdr.EnvelopeTypeEnvelopeTypeTxFeeBump:
		feeBumpAccount := t.Envelope.FeeBumpAccount()
		feeAccountAddress := feeBumpAccount.Address()
		return feeAccountAddress, true
	default:
		return "", false
	}
}

func (t *LedgerTransaction) SorobanInclusionFeeCharged() (int64, bool) {
	resourceFee, ok := t.SorobanResourceFee()
	if !ok {
		return 0, false
	}

	feeAccountAddress, ok := t.FeeAccountAddress()
	if !ok {
		return 0, false
	}

	accountBalanceStart, accountBalanceEnd := getAccountBalanceFromLedgerEntryChanges(t.FeeChanges, feeAccountAddress)
	initialFeeCharged := accountBalanceStart - accountBalanceEnd

	return initialFeeCharged - resourceFee, true
}

func getAccountBalanceFromLedgerEntryChanges(changes xdr.LedgerEntryChanges, sourceAccountAddress string) (int64, int64) {
	var accountBalanceStart int64
	var accountBalanceEnd int64

	for _, change := range changes {
		switch change.Type {
		case xdr.LedgerEntryChangeTypeLedgerEntryUpdated:
			accountEntry, ok := change.Updated.Data.GetAccount()
			if !ok {
				continue
			}

			if accountEntry.AccountId.Address() == sourceAccountAddress {
				accountBalanceEnd = int64(accountEntry.Balance)
			}
		case xdr.LedgerEntryChangeTypeLedgerEntryState:
			accountEntry, ok := change.State.Data.GetAccount()
			if !ok {
				continue
			}

			if accountEntry.AccountId.Address() == sourceAccountAddress {
				accountBalanceStart = int64(accountEntry.Balance)
			}
		}
	}

	return accountBalanceStart, accountBalanceEnd
}

func (t *LedgerTransaction) SorobanResourceFeeRefund() (int64, bool) {
	meta, ok := t.UnsafeMeta.GetV3()
	if !ok {
		return 0, false
	}

	feeAccountAddress, ok := t.FeeAccountAddress()
	if !ok {
		return 0, false
	}

	accountBalanceStart, accountBalanceEnd := getAccountBalanceFromLedgerEntryChanges(meta.TxChangesAfter, feeAccountAddress)

	return accountBalanceEnd - accountBalanceStart, true

}

func (t *LedgerTransaction) SorobanTotalNonRefundableResourceFeeCharged() (int64, bool) {
	meta, ok := t.UnsafeMeta.GetV3()
	if !ok {
		return 0, false
	}

	switch meta.SorobanMeta.Ext.V {
	case 1:
		return int64(meta.SorobanMeta.Ext.V1.TotalNonRefundableResourceFeeCharged), true
	default:
		return 0, false
	}
}

func (t *LedgerTransaction) SorobanTotalRefundableResourceFeeCharged() (int64, bool) {
	meta, ok := t.UnsafeMeta.GetV3()
	if !ok {
		return 0, false
	}

	switch meta.SorobanMeta.Ext.V {
	case 1:
		return int64(meta.SorobanMeta.Ext.V1.TotalRefundableResourceFeeCharged), true
	default:
		return 0, false
	}
}

func (t *LedgerTransaction) SorobanRentFeeCharged() (int64, bool) {
	meta, ok := t.UnsafeMeta.GetV3()
	if !ok {
		return 0, false
	}

	switch meta.SorobanMeta.Ext.V {
	case 1:
		return int64(meta.SorobanMeta.Ext.V1.RentFeeCharged), true
	default:
		return 0, false
	}
}

func (t *LedgerTransaction) ResultCode() string {
	return t.Result.Result.Result.Code.String()
}

func (t *LedgerTransaction) Signers() (signers []string, err error) {
	if t.Envelope.IsFeeBump() {
		return getTxSigners(t.Envelope.FeeBump.Signatures)
	}

	return getTxSigners(t.Envelope.Signatures())
}

func getTxSigners(xdrSignatures []xdr.DecoratedSignature) ([]string, error) {
	signers := make([]string, len(xdrSignatures))

	for i, sig := range xdrSignatures {
		signerAccount, err := strkey.Encode(strkey.VersionByteAccountID, sig.Signature)
		if err != nil {
			return nil, err
		}
		signers[i] = signerAccount
	}

	return signers, nil
}

func (t *LedgerTransaction) AccountMuxed() (string, bool) {
	sourceAccount := t.Envelope.SourceAccount()
	if sourceAccount.Type != xdr.CryptoKeyTypeKeyTypeMuxedEd25519 {
		return "", false
	}

	return sourceAccount.Address(), true

}

func (t *LedgerTransaction) FeeAccount() (string, bool) {
	if !t.Envelope.IsFeeBump() {
		return "", false
	}

	feeBumpAccount := t.Envelope.FeeBumpAccount()
	feeAccount := feeBumpAccount.ToAccountId()

	return feeAccount.Address(), true

}

func (t *LedgerTransaction) FeeAccountMuxed() (string, bool) {
	if !t.Envelope.IsFeeBump() {
		return "", false
	}

	feeBumpAccount := t.Envelope.FeeBumpAccount()
	if feeBumpAccount.Type != xdr.CryptoKeyTypeKeyTypeMuxedEd25519 {
		return "", false
	}

	return feeBumpAccount.Address(), true
}

func (t *LedgerTransaction) InnerTransactionHash() (string, bool) {
	if !t.Envelope.IsFeeBump() {
		return "", false
	}

	innerHash := t.Result.InnerHash()

	return hex.EncodeToString(innerHash[:]), true
}

func (t *LedgerTransaction) NewMaxFee() (uint32, bool) {
	if !t.Envelope.IsFeeBump() {
		return 0, false
	}

	return uint32(t.Envelope.FeeBumpFee()), true
}

func (t *LedgerTransaction) Successful() bool {
	return t.Result.Successful()
}

func (t *LedgerTransaction) GetOperations(networkPassphrase string) []LedgerOperation {
	var ledgerOperations []LedgerOperation

	for i, operation := range t.Envelope.Operations() {
		ledgerOperation := LedgerOperation{
			OperationIndex:    int32(i),
			Operation:         operation,
			Transaction:       t,
			NetworkPassphrase: networkPassphrase,
		}
		ledgerOperations = append(ledgerOperations, ledgerOperation)
	}

	return ledgerOperations
}

func (t *LedgerTransaction) GetTransactionV1Envelope() (xdr.TransactionV1Envelope, bool) {
	switch t.Envelope.Type {
	case xdr.EnvelopeTypeEnvelopeTypeTx:
		switch t.Envelope.Type {
		case 2:
			return *t.Envelope.V1, true
		default:
			return xdr.TransactionV1Envelope{}, false
		}
	case xdr.EnvelopeTypeEnvelopeTypeTxFeeBump:
		return t.Envelope.MustFeeBump().Tx.InnerTx.MustV1(), true
	default:
		return xdr.TransactionV1Envelope{}, false
	}
}

func (t *LedgerTransaction) LedgerKeyHashFromTxEnvelope() []string {
	var ledgerKeyHash []string

	v1Envelope, ok := t.GetTransactionV1Envelope()
	if !ok {
		return ledgerKeyHash
	}

	for _, ledgerKey := range v1Envelope.Tx.Ext.SorobanData.Resources.Footprint.ReadOnly {
		ledgerKeyBase64, err := ledgerKey.MarshalBinaryBase64()
		if err != nil {
			panic(err)
		}
		if ledgerKeyBase64 != "" {
			ledgerKeyHash = append(ledgerKeyHash, ledgerKeyBase64)
		}
	}

	for _, ledgerKey := range v1Envelope.Tx.Ext.SorobanData.Resources.Footprint.ReadWrite {
		ledgerKeyBase64, err := ledgerKey.MarshalBinaryBase64()
		if err != nil {
			panic(err)
		}
		if ledgerKeyBase64 != "" {
			ledgerKeyHash = append(ledgerKeyHash, ledgerKeyBase64)
		}
	}

	return ledgerKeyHash
}

func (t *LedgerTransaction) ContractCodeHashFromTxEnvelope() (string, bool) {
	v1Envelope, ok := t.GetTransactionV1Envelope()
	if !ok {
		return "", false
	}
	for _, ledgerKey := range v1Envelope.Tx.Ext.SorobanData.Resources.Footprint.ReadOnly {
		contractCode, ok := t.contractCodeFromContractData(ledgerKey)
		if !ok {
			return "", false
		}
		if contractCode != "" {
			return contractCode, true
		}
	}

	for _, ledgerKey := range v1Envelope.Tx.Ext.SorobanData.Resources.Footprint.ReadWrite {
		contractCode, ok := t.contractCodeFromContractData(ledgerKey)
		if !ok {
			return "", false
		}
		if contractCode != "" {
			return contractCode, true
		}
	}

	return "", true
}

func (t *LedgerTransaction) contractCodeFromContractData(ledgerKey xdr.LedgerKey) (string, bool) {
	contractCode, ok := ledgerKey.GetContractCode()
	if !ok {
		return "", false
	}

	codeHash, err := contractCode.Hash.MarshalBinaryBase64()
	if err != nil {
		panic(err)
	}

	return codeHash, true
}

func (t *LedgerTransaction) contractIdFromTxEnvelope() (string, bool) {
	v1Envelope, ok := t.GetTransactionV1Envelope()
	if !ok {
		return "", false
	}
	for _, ledgerKey := range v1Envelope.Tx.Ext.SorobanData.Resources.Footprint.ReadWrite {
		contractId, ok := t.contractIdFromContractData(ledgerKey)
		if !ok {
			return "", false
		}

		if contractId != "" {
			return contractId, true
		}
	}

	for _, ledgerKey := range v1Envelope.Tx.Ext.SorobanData.Resources.Footprint.ReadOnly {
		contractId, ok := t.contractIdFromContractData(ledgerKey)
		if !ok {
			return "", false
		}

		if contractId != "" {
			return contractId, true
		}
	}

	return "", true
}

func (t *LedgerTransaction) contractIdFromContractData(ledgerKey xdr.LedgerKey) (string, bool) {
	contractData, ok := ledgerKey.GetContractData()
	if !ok {
		return "", false
	}
	contractIdHash, ok := contractData.Contract.GetContractId()
	if !ok {
		return "", false
	}

	var contractIdByte []byte
	var contractId string
	var err error
	contractIdByte, err = contractIdHash.MarshalBinary()
	if err != nil {
		panic(err)
	}
	contractId, err = strkey.Encode(strkey.VersionByteContract, contractIdByte)
	if err != nil {
		panic(err)
	}

	return contractId, true
}
