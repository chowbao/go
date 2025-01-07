package xdrill

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strconv"
	"time"

	"github.com/stellar/go/exp/xdrill/utils"
	"github.com/stellar/go/ingest"
	"github.com/stellar/go/toid"
	"github.com/stellar/go/xdr"
)

type Transaction struct {
	transaction *ingest.LedgerTransaction
	ledger      *Ledger
}

func (t Transaction) Sequence() uint32 {
	return t.ledger.Sequence()
}

func (t Transaction) Hash() string {
	return utils.HashToHexString(t.transaction.Result.TransactionHash)
}

func (t Transaction) Index() uint32 {
	return uint32(t.transaction.Index)
}

func (t Transaction) ID() int64 {
	return toid.New(int32(t.Sequence()), int32(t.Index()), 0).ToInt64()
}

func (t Transaction) Account() (string, error) {
	return utils.GetAccountAddressFromMuxedAccount(t.transaction.Envelope.SourceAccount())
}

func (t Transaction) AccountSequence() int64 {
	return t.transaction.Envelope.SeqNum()
}

func (t Transaction) MaxFee() uint32 {
	return t.transaction.Envelope.Fee()
}

func (t Transaction) FeeCharged() int64 {
	// Any Soroban Fee Bump transactions before P21 will need the below logic to calculate the correct feeCharged
	// Protocol 20 contained a bug where the feeCharged was incorrectly calculated but was fixed for
	// Protocol 21 with https://github.com/stellar/stellar-core/issues/4188
	_, ok := t.getSorobanData()
	if ok {
		if t.ledger.LedgerVersion() < 21 && t.transaction.Envelope.Type == xdr.EnvelopeTypeEnvelopeTypeTxFeeBump {
			return int64(t.transaction.Result.Result.FeeCharged) - t.SorobanResourceFeeRefund() + t.SorobanInclusionFeeCharged()
		}
	}

	return int64(t.transaction.Result.Result.FeeCharged)
}

func (t Transaction) OperationCount() uint32 {
	return uint32(len(t.transaction.Envelope.Operations()))
}

func (t Transaction) ClosedAt() time.Time {
	return t.ledger.ClosedAt()
}

func (t Transaction) Memo() string {
	memoObject := t.transaction.Envelope.Memo()
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

func (t Transaction) MemmoType() string {
	memoObject := t.transaction.Envelope.Memo()
	return memoObject.Type.String()
}

func (t Transaction) TimeBounds() (string, error) {
	timeBounds := t.transaction.Envelope.TimeBounds()
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

func (t Transaction) LedgerBounds() string {
	ledgerBounds := t.transaction.Envelope.LedgerBounds()
	if ledgerBounds == nil {
		return ""
	}

	return fmt.Sprintf("[%d,%d)", int64(ledgerBounds.MinLedger), int64(ledgerBounds.MaxLedger))
}

func (t Transaction) MinSequence() int64 {
	minSequenceNumber := t.transaction.Envelope.MinSeqNum()
	if minSequenceNumber == nil {
		return 0
	}

	return int64(*minSequenceNumber)
}

func (t Transaction) MinSequenceAge() int64 {
	minSequenceAge := t.transaction.Envelope.MinSeqAge()
	if minSequenceAge == nil {
		return 0
	}

	return int64(*minSequenceAge)
}

func (t Transaction) MinSequenceLedgerGap() int64 {
	minSequenceLedgerGap := t.transaction.Envelope.MinSeqLedgerGap()
	if minSequenceLedgerGap == nil {
		return 0
	}

	return int64(*minSequenceLedgerGap)
}

func (t Transaction) getSorobanData() (sorobanData xdr.SorobanTransactionData, ok bool) {
	switch t.transaction.Envelope.Type {
	case xdr.EnvelopeTypeEnvelopeTypeTx:
		sorobanData, ok = t.transaction.Envelope.V1.Tx.Ext.GetSorobanData()
	case xdr.EnvelopeTypeEnvelopeTypeTxFeeBump:
		sorobanData, ok = t.transaction.Envelope.FeeBump.Tx.InnerTx.V1.Tx.Ext.GetSorobanData()
	}

	return
}

func (t Transaction) SorobanResourceFee() int64 {
	sorobanData, ok := t.getSorobanData()
	if !ok {
		return 0
	}

	return int64(sorobanData.ResourceFee)
}

func (t Transaction) SorobanResourcesInstructions() uint32 {
	sorobanData, ok := t.getSorobanData()
	if !ok {
		return 0
	}

	return uint32(sorobanData.Resources.Instructions)
}

func (t Transaction) SorobanResourcesReadBytes() uint32 {
	sorobanData, ok := t.getSorobanData()
	if !ok {
		return 0
	}

	return uint32(sorobanData.Resources.ReadBytes)
}

func (t Transaction) SorobanResourcesWriteBytes() uint32 {
	sorobanData, ok := t.getSorobanData()
	if !ok {
		return 0
	}

	return uint32(sorobanData.Resources.WriteBytes)
}

func (t Transaction) InclusionFeeBid() int64 {
	return int64(t.transaction.Envelope.Fee()) - t.SorobanResourceFee()
}

func (t Transaction) getFeeAccountAddress() (feeAccountAddress string) {
	switch t.transaction.Envelope.Type {
	case xdr.EnvelopeTypeEnvelopeTypeTx:
		sourceAccount := t.transaction.Envelope.SourceAccount()
		feeAccountAddress = sourceAccount.Address()
	case xdr.EnvelopeTypeEnvelopeTypeTxFeeBump:
		feeBumpAccount := t.transaction.Envelope.FeeBumpAccount()
		feeAccountAddress = feeBumpAccount.Address()
	}

	return
}

func (t Transaction) SorobanInclusionFeeCharged() int64 {
	accountBalanceStart, accountBalanceEnd := utils.GetAccountBalanceFromLedgerEntryChanges(t.transaction.FeeChanges, t.getFeeAccountAddress())
	initialFeeCharged := accountBalanceStart - accountBalanceEnd
	return initialFeeCharged - t.SorobanResourceFee()
}

func (t Transaction) SorobanResourceFeeRefund() int64 {
	meta, ok := t.transaction.UnsafeMeta.GetV3()
	if !ok {
		return 0
	}

	accountBalanceStart, accountBalanceEnd := utils.GetAccountBalanceFromLedgerEntryChanges(meta.TxChangesAfter, t.getFeeAccountAddress())
	return accountBalanceEnd - accountBalanceStart
}

func (t Transaction) SorobanTotalNonRefundableResourceFeeCharged() int64 {
	meta, ok := t.transaction.UnsafeMeta.GetV3()
	if !ok {
		return 0
	}

	switch meta.SorobanMeta.Ext.V {
	case 1:
		return int64(meta.SorobanMeta.Ext.V1.TotalNonRefundableResourceFeeCharged)
	default:
		return 0
	}
}

func (t Transaction) SorobanTotalRefundableResourceFeeCharged() int64 {
	meta, ok := t.transaction.UnsafeMeta.GetV3()
	if !ok {
		return 0
	}

	switch meta.SorobanMeta.Ext.V {
	case 1:
		return int64(meta.SorobanMeta.Ext.V1.TotalRefundableResourceFeeCharged)
	default:
		return 0
	}
}

func (t Transaction) SorobanRentFeeCharged() int64 {
	meta, ok := t.transaction.UnsafeMeta.GetV3()
	if !ok {
		return 0
	}

	switch meta.SorobanMeta.Ext.V {
	case 1:
		return int64(meta.SorobanMeta.Ext.V1.RentFeeCharged)
	default:
		return 0
	}
}

func (t Transaction) ResultCode() string {
	return t.transaction.Result.Result.Result.Code.String()
}

func (t Transaction) Signers() (signers []string) {
	if t.transaction.Envelope.IsFeeBump() {
		signers, _ = utils.GetTxSigners(t.transaction.Envelope.FeeBump.Signatures)
		return
	}

	signers, _ = utils.GetTxSigners(t.transaction.Envelope.Signatures())
	return
}

func (t Transaction) AccountMuxed() string {
	sourceAccount := t.transaction.Envelope.SourceAccount()
	if sourceAccount.Type != xdr.CryptoKeyTypeKeyTypeMuxedEd25519 {
		return ""
	}

	return sourceAccount.Address()
}

func (t Transaction) FeeAccount() string {
	if !t.transaction.Envelope.IsFeeBump() {
		return ""
	}

	feeBumpAccount := t.transaction.Envelope.FeeBumpAccount()
	feeAccount := feeBumpAccount.ToAccountId()

	return feeAccount.Address()
}

func (t Transaction) FeeAccountMuxed() string {
	if !t.transaction.Envelope.IsFeeBump() {
		return ""
	}

	feeBumpAccount := t.transaction.Envelope.FeeBumpAccount()
	if feeBumpAccount.Type != xdr.CryptoKeyTypeKeyTypeMuxedEd25519 {
		return ""
	}

	return feeBumpAccount.Address()
}

func (t Transaction) InnerTransactionHash() string {
	if !t.transaction.Envelope.IsFeeBump() {
		return ""
	}

	innerHash := t.transaction.Result.InnerHash()
	return hex.EncodeToString(innerHash[:])
}

func (t Transaction) NewMaxFee() uint32 {
	return uint32(t.transaction.Envelope.FeeBumpFee())
}

func (t Transaction) Successful() bool {
	return t.transaction.Result.Successful()
}
