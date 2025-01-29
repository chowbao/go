package ingest

import (
	"bytes"
	"fmt"
	"math/big"
	"sort"
	"time"

	"github.com/stellar/go/ingest/ledger"
	"github.com/stellar/go/strkey"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

// Change is a developer friendly representation of LedgerEntryChanges.
// It also provides some helper functions to quickly check if a given
// change has occurred in an entry.
//
// Change represents a modification to a ledger entry, capturing both the before and after states
// of the entry along with the context that explains what caused the change. It is primarily used to
// track changes during transactions and/or operations within a transaction
// and can be helpful in identifying the specific cause of changes to the LedgerEntry state. (https://github.com/stellar/go/issues/5535
//
// Behavior:
//
//   - **Created entries**: Pre is nil, and Post is not nil.
//
//   - **Updated entries**: Both Pre and Post are non-nil.
//
//   - **Removed entries**: Pre is not nil, and Post is nil.
//
//     A `Change` can be caused primarily by either a transaction or by an operation within a transaction:
//
//   - **Operations**:
//     Each successful operation can cause multiple ledger entry changes.
//     For example, a path payment operation may affect the source and destination account entries,
//     as well as potentially modify offers and/or liquidity pools.
//
//   - **Transactions**:
//     Some ledger changes, such as those involving fees or account balances, may be caused by
//     the transaction itself and may not be tied to a specific operation within a transaction.
//     For instance, fees for all operations in a transaction are debited from the source account,
//     triggering ledger changes without operation-specific details.
type Change struct {
	// The type of the ledger entry being changed.
	Type xdr.LedgerEntryType

	// The state of the LedgerEntry before the change. This will be nil if the entry was created.
	Pre *xdr.LedgerEntry

	// The state of the LedgerEntry after the change. This will be nil if the entry was removed.
	Post *xdr.LedgerEntry

	// Specifies why the change occurred, represented as a LedgerEntryChangeReason
	Reason LedgerEntryChangeReason

	// The index of the operation within the transaction that caused the change.
	// This field is relevant only when the Reason is LedgerEntryChangeReasonOperation
	// This field cannot be relied upon when the compactingChangeReader is used.
	OperationIndex uint32

	// The LedgerTransaction responsible for the change.
	// It contains details such as transaction hash, envelope, result pair, and fees.
	// This field is populated only when the Reason is one of:
	// LedgerEntryChangeReasonTransaction, LedgerEntryChangeReasonOperation or LedgerEntryChangeReasonFee
	Transaction *LedgerTransaction

	// The LedgerCloseMeta that precipitated the change.
	// This is useful only when the Change is caused by an upgrade or by an eviction, i.e. outside a transaction
	// This field is populated only when the Reason is one of:
	// LedgerEntryChangeReasonUpgrade or LedgerEntryChangeReasonEviction
	// For changes caused by transaction or operations, look at the Transaction field
	Ledger *xdr.LedgerCloseMeta

	// Information about the upgrade, if the change occurred as part of an upgrade
	// This field is relevant only when the Reason is LedgerEntryChangeReasonUpgrade
	LedgerUpgrade *xdr.LedgerUpgrade
}

// LedgerEntryChangeReason represents the reason for a ledger entry change.
type LedgerEntryChangeReason uint16

const (
	// LedgerEntryChangeReasonUnknown indicates an unknown or unsupported change reason
	LedgerEntryChangeReasonUnknown LedgerEntryChangeReason = iota

	// LedgerEntryChangeReasonOperation indicates a change caused by an operation in a transaction
	LedgerEntryChangeReasonOperation

	// LedgerEntryChangeReasonTransaction indicates a change caused by the transaction itself
	LedgerEntryChangeReasonTransaction

	// LedgerEntryChangeReasonFee indicates a change related to transaction fees.
	LedgerEntryChangeReasonFee

	// LedgerEntryChangeReasonUpgrade indicates a change caused by a ledger upgrade.
	LedgerEntryChangeReasonUpgrade

	// LedgerEntryChangeReasonEviction indicates a change caused by entry eviction.
	LedgerEntryChangeReasonEviction
)

// String returns a best effort string representation of the change.
// If the Pre or Post xdr is invalid, the field will be omitted from the string.
func (c Change) String() string {
	var pre, post string
	if c.Pre != nil {
		if b64, err := xdr.MarshalBase64(c.Pre); err == nil {
			pre = b64
		}
	}
	if c.Post != nil {
		if b64, err := xdr.MarshalBase64(c.Post); err == nil {
			post = b64
		}
	}
	return fmt.Sprintf(
		"Change{Type: %s, Pre: %s, Post: %s}",
		c.Type.String(),
		pre,
		post,
	)
}

func (c Change) LedgerKey() (xdr.LedgerKey, error) {
	if c.Pre != nil {
		return c.Pre.LedgerKey()
	}
	return c.Post.LedgerKey()
}

// GetChangesFromLedgerEntryChanges transforms LedgerEntryChanges to []Change.
// Each `update` and `removed` is preceded with `state` and `create` changes
// are alone, without `state`. The transformation we're doing is to move each
// change (state/update, state/removed or create) to an array of pre/post pairs.
// Then:
// - for create, pre is null and post is a new entry,
// - for update, pre is previous state and post is the current state,
// - for removed, pre is previous state and post is null.
//
// stellar-core source:
// https://github.com/stellar/stellar-core/blob/e584b43/src/ledger/LedgerTxn.cpp#L582
func GetChangesFromLedgerEntryChanges(ledgerEntryChanges xdr.LedgerEntryChanges) []Change {
	changes := make([]Change, 0, len(ledgerEntryChanges))
	for i, entryChange := range ledgerEntryChanges {
		switch entryChange.Type {
		case xdr.LedgerEntryChangeTypeLedgerEntryCreated:
			created := entryChange.MustCreated()
			changes = append(changes, Change{
				Type: created.Data.Type,
				Pre:  nil,
				Post: &created,
			})
		case xdr.LedgerEntryChangeTypeLedgerEntryUpdated:
			state := ledgerEntryChanges[i-1].MustState()
			updated := entryChange.MustUpdated()
			changes = append(changes, Change{
				Type: state.Data.Type,
				Pre:  &state,
				Post: &updated,
			})
		case xdr.LedgerEntryChangeTypeLedgerEntryRemoved:
			state := ledgerEntryChanges[i-1].MustState()
			changes = append(changes, Change{
				Type: state.Data.Type,
				Pre:  &state,
				Post: nil,
			})
		case xdr.LedgerEntryChangeTypeLedgerEntryState:
			continue
		default:
			panic("Invalid LedgerEntryChangeType")
		}
	}

	sortChanges(changes)
	return changes
}

type sortableChanges struct {
	changes    []Change
	ledgerKeys [][]byte
}

func newSortableChanges(changes []Change) sortableChanges {
	ledgerKeys := make([][]byte, len(changes))
	for i, c := range changes {
		lk, err := c.LedgerKey()
		if err != nil {
			panic(err)
		}
		lkBytes, err := lk.MarshalBinary()
		if err != nil {
			panic(err)
		}
		ledgerKeys[i] = lkBytes
	}
	return sortableChanges{
		changes:    changes,
		ledgerKeys: ledgerKeys,
	}
}

func (s sortableChanges) Len() int {
	return len(s.changes)
}

func (s sortableChanges) Less(i, j int) bool {
	return bytes.Compare(s.ledgerKeys[i], s.ledgerKeys[j]) < 0
}

func (s sortableChanges) Swap(i, j int) {
	s.changes[i], s.changes[j] = s.changes[j], s.changes[i]
	s.ledgerKeys[i], s.ledgerKeys[j] = s.ledgerKeys[j], s.ledgerKeys[i]
}

// sortChanges is applied on a list of changes to ensure that LedgerEntryChanges
// from Tx Meta are ingested in a deterministic order.
// The changes are sorted by ledger key. It is unexpected for there to be
// multiple changes with the same ledger key in a LedgerEntryChanges group,
// but if that is the case, we fall back to the original ordering of the changes
// by using a stable sorting algorithm.
func sortChanges(changes []Change) {
	sort.Stable(newSortableChanges(changes))
}

// LedgerEntryChangeType returns type in terms of LedgerEntryChangeType.
func (c Change) LedgerEntryChangeType() xdr.LedgerEntryChangeType {
	switch {
	case c.Pre == nil && c.Post != nil:
		return xdr.LedgerEntryChangeTypeLedgerEntryCreated
	case c.Pre != nil && c.Post == nil:
		return xdr.LedgerEntryChangeTypeLedgerEntryRemoved
	case c.Pre != nil && c.Post != nil:
		return xdr.LedgerEntryChangeTypeLedgerEntryUpdated
	default:
		panic("Invalid state of Change (Pre == nil && Post == nil)")
	}
}

// getLiquidityPool gets the most recent state of the LiquidityPool that exists or existed.
func (c Change) getLiquidityPool() (*xdr.LiquidityPoolEntry, error) {
	var entry *xdr.LiquidityPoolEntry
	if c.Pre != nil {
		entry = c.Pre.Data.LiquidityPool
	}
	if c.Post != nil {
		entry = c.Post.Data.LiquidityPool
	}
	if entry == nil {
		return &xdr.LiquidityPoolEntry{}, errors.New("this change does not include a liquidity pool")
	}
	return entry, nil
}

// GetLiquidityPoolType returns the liquidity pool type.
func (c Change) GetLiquidityPoolType() (xdr.LiquidityPoolType, error) {
	lp, err := c.getLiquidityPool()
	if err != nil {
		return xdr.LiquidityPoolType(0), err
	}
	return lp.Body.Type, nil
}

// AccountChangedExceptSigners returns true if account has changed WITHOUT
// checking the signers (except master key weight!). In other words, if the only
// change is connected to signers, this function will return false.
func (c Change) AccountChangedExceptSigners() (bool, error) {
	if c.Type != xdr.LedgerEntryTypeAccount {
		panic("This should not be called on changes other than Account changes")
	}

	// New account
	if c.Pre == nil {
		return true, nil
	}

	// Account merged
	// c.Pre != nil at this point.
	if c.Post == nil {
		return true, nil
	}

	// c.Pre != nil && c.Post != nil at this point.
	if c.Pre.LastModifiedLedgerSeq != c.Post.LastModifiedLedgerSeq {
		return true, nil
	}

	// Don't use short assignment statement (:=) to ensure variables below
	// are not pointers (if `xdr` package changes in the future)!
	var preAccountEntry, postAccountEntry xdr.AccountEntry
	preAccountEntry = c.Pre.Data.MustAccount()
	postAccountEntry = c.Post.Data.MustAccount()

	// preAccountEntry and postAccountEntry are copies so it's fine to
	// modify them here, EXCEPT pointers inside them!
	if preAccountEntry.Ext.V == 0 {
		preAccountEntry.Ext.V = 1
		preAccountEntry.Ext.V1 = &xdr.AccountEntryExtensionV1{
			Liabilities: xdr.Liabilities{
				Buying:  0,
				Selling: 0,
			},
		}
	}

	preAccountEntry.Signers = nil

	if postAccountEntry.Ext.V == 0 {
		postAccountEntry.Ext.V = 1
		postAccountEntry.Ext.V1 = &xdr.AccountEntryExtensionV1{
			Liabilities: xdr.Liabilities{
				Buying:  0,
				Selling: 0,
			},
		}
	}

	postAccountEntry.Signers = nil

	preBinary, err := preAccountEntry.MarshalBinary()
	if err != nil {
		return false, errors.Wrap(err, "Error running preAccountEntry.MarshalBinary")
	}

	postBinary, err := postAccountEntry.MarshalBinary()
	if err != nil {
		return false, errors.Wrap(err, "Error running postAccountEntry.MarshalBinary")
	}

	return !bytes.Equal(preBinary, postBinary), nil
}

func (c Change) ExtractEntry() (xdr.LedgerEntry, xdr.LedgerEntryChangeType, bool, error) {
	switch changeType := c.LedgerEntryChangeType(); changeType {
	case xdr.LedgerEntryChangeTypeLedgerEntryCreated, xdr.LedgerEntryChangeTypeLedgerEntryUpdated:
		return *c.Post, changeType, false, nil
	case xdr.LedgerEntryChangeTypeLedgerEntryRemoved:
		return *c.Pre, changeType, true, nil
	default:
		return xdr.LedgerEntry{}, changeType, false, fmt.Errorf("unable to extract ledger entry type from change")
	}
}

func (c Change) Deleted() (bool, error) {
	_, _, deleted, err := c.ExtractEntry()
	if err != nil {
		return false, err
	}

	return deleted, nil
}

func (c Change) ClosedAt() time.Time {
	if c.Ledger != nil {
		return ledger.ClosedAt(*c.Ledger)
	}

	return ledger.ClosedAt(c.Transaction.Ledger)
}

func (c Change) Sequence() uint32 {
	if c.Ledger != nil {
		return ledger.Sequence(*c.Ledger)
	}

	return ledger.Sequence(c.Transaction.Ledger)
}

func (c Change) LastModifiedLedger() (uint32, error) {
	ledgerEntry, _, _, err := c.ExtractEntry()
	if err != nil {
		return 0, err
	}

	return uint32(ledgerEntry.LastModifiedLedgerSeq), nil
}

func (c Change) Sponsor() (string, error) {
	ledgerEntry, _, _, err := c.ExtractEntry()
	if err != nil {
		return "", err
	}

	if ledgerEntry.SponsoringID() == nil {
		return "", nil
	}

	return ledgerEntry.SponsoringID().Address(), nil
}

func (c Change) LedgerKeyHash() (string, error) {
	ledgerKey, err := c.LedgerKey()
	if err != nil {
		return "", err
	}

	return ledgerKey.MarshalBinaryBase64()
}

func (c Change) EntryDetails(passphrase string) (map[string]interface{}, error) {
	var err error
	var ledgerEntry xdr.LedgerEntry
	var details map[string]interface{}

	ledgerEntry, _, _, err = c.ExtractEntry()
	if err != nil {
		return nil, err
	}

	switch ledgerEntry.Data.Type {
	case xdr.LedgerEntryTypeAccount:
		details, err = AccountDetails(ledgerEntry.Data.Account)
		if err != nil {
			return details, err
		}
	case xdr.LedgerEntryTypeTrustline:
		details, err = TrustlineDetails(ledgerEntry.Data.TrustLine)
		if err != nil {
			return details, err
		}
	case xdr.LedgerEntryTypeOffer:
		details, err = OfferDetails(ledgerEntry.Data.Offer)
		if err != nil {
			return details, err
		}
	case xdr.LedgerEntryTypeData:
		details, err = DataDetails(ledgerEntry.Data.Data)
		if err != nil {
			return details, err
		}
	case xdr.LedgerEntryTypeClaimableBalance:
		details, err = ClaimableBalanceDetails(ledgerEntry.Data.ClaimableBalance)
		if err != nil {
			return details, err
		}
	case xdr.LedgerEntryTypeLiquidityPool:
		details, err = LiquidityPoolDetails(ledgerEntry.Data.LiquidityPool)
		if err != nil {
			return details, err
		}
	case xdr.LedgerEntryTypeContractData:
		details, err = ContractDataDetails(passphrase, ledgerEntry.Data.ContractData)
		if err != nil {
			return details, err
		}
	case xdr.LedgerEntryTypeContractCode:
		details, err = ContractCodeDetails(ledgerEntry.Data.ContractCode)
		if err != nil {
			return details, err
		}
	case xdr.LedgerEntryTypeTtl:
		details, err = TtlDetails(ledgerEntry.Data.Ttl)
		if err != nil {
			return details, err
		}
	default:
		return details, fmt.Errorf("unknown LedgerEntry data type")
	}

	return details, nil
}

type Signers struct {
	Address string
	Weight  int32
	Sponsor string
}

func AccountDetails(accountEntry *xdr.AccountEntry) (map[string]interface{}, error) {
	details := map[string]interface{}{}

	details["account_id"] = accountEntry.AccountId.Address()
	details["balance"] = int64(accountEntry.Balance)
	details["sequence_number"] = int64(accountEntry.SeqNum)
	details["sequence_ledger"] = uint32(accountEntry.SeqLedger())
	details["sequence_time"] = int64(accountEntry.SeqTime())
	details["num_sub_entries"] = uint32(accountEntry.NumSubEntries)
	details["flags"] = uint32(accountEntry.Flags)
	details["home_domain"] = accountEntry.HomeDomain
	details["master_key_weight"] = int32(accountEntry.MasterKeyWeight())
	details["threshold_low"] = int32(accountEntry.ThresholdLow())
	details["threshold_medium"] = int32(accountEntry.ThresholdMedium())
	details["threshold_high"] = int32(accountEntry.ThresholdHigh())
	details["num_sponsored"] = uint32(accountEntry.NumSponsored())
	details["num_sponsoring"] = uint32(accountEntry.NumSponsoring())

	if accountEntry.InflationDest != nil {
		details["inflation_destination"] = accountEntry.InflationDest.Address()
	}

	accountExtensionInfo, ok := accountEntry.Ext.GetV1()
	if ok {
		details["buying_liabilities"] = int64(accountExtensionInfo.Liabilities.Buying)
		details["selling_liabilities"] = int64(accountExtensionInfo.Liabilities.Selling)
	}

	signers := []Signers{}
	sponsors := accountEntry.SponsorPerSigner()
	for signer, weight := range accountEntry.SignerSummary() {
		sponsorDesc := sponsors[signer]

		signers = append(signers, Signers{
			Address: signer,
			Weight:  weight,
			Sponsor: sponsorDesc.Address(),
		})
	}

	details["signers"] = signers

	return details, nil
}

func TrustlineDetails(trustlineEntry *xdr.TrustLineEntry) (map[string]interface{}, error) {
	details := map[string]interface{}{}

	details["account_id"] = trustlineEntry.AccountId.Address()
	details["balance"] = int64(trustlineEntry.Balance)
	details["trustline_limit"] = int64(trustlineEntry.Limit)
	details["buying_liabilities"] = int64(trustlineEntry.Liabilities().Buying)
	details["selling_liabilities"] = int64(trustlineEntry.Liabilities().Selling)
	details["flags"] = uint32(trustlineEntry.Flags)

	var err error
	var assetType, assetCode, assetIssuer string
	err = trustlineEntry.Asset.Extract(&assetType, &assetCode, &assetIssuer)
	if err != nil {
		return details, err
	}

	details["asset_code"] = assetCode
	details["asset_issuer"] = assetIssuer
	details["asset_type"] = assetType

	var poolID string
	poolID, err = trustlineEntry.Asset.LiquidityPoolId.MarshalBinaryBase64()
	if err != nil {
		return details, err
	}

	details["liquidity_pool_id"] = poolID

	return details, nil
}

func OfferDetails(offerEntry *xdr.OfferEntry) (map[string]interface{}, error) {
	details := map[string]interface{}{}

	details["seller_id"] = offerEntry.SellerId.Address()
	details["offer_id"] = int64(offerEntry.OfferId)
	details["amount"] = int64(offerEntry.Amount)
	details["price_n"] = int32(offerEntry.Price.N)
	details["price_d"] = int32(offerEntry.Price.D)
	details["flags"] = uint32(offerEntry.Flags)

	var err error
	var sellingAssetType, sellingAssetCode, sellingAssetIssuer string
	err = offerEntry.Selling.Extract(&sellingAssetType, &sellingAssetCode, &sellingAssetIssuer)
	if err != nil {
		return details, err
	}

	details["selling_asset_code"] = sellingAssetCode
	details["selling_asset_issuer"] = sellingAssetIssuer
	details["selling_asset_type"] = sellingAssetType

	var buyingAssetType, buyingAssetCode, buyingAssetIssuer string
	err = offerEntry.Buying.Extract(&buyingAssetType, &buyingAssetCode, &buyingAssetIssuer)
	if err != nil {
		return details, err
	}

	details["buying_asset_code"] = buyingAssetCode
	details["buying_asset_issuer"] = buyingAssetIssuer
	details["buying_asset_type"] = buyingAssetType

	return details, nil
}

func DataDetails(dataEntry *xdr.DataEntry) (map[string]interface{}, error) {
	details := map[string]interface{}{}

	details["account_id"] = dataEntry.AccountId.Address()
	details["data_name"] = string(dataEntry.DataName)

	dataValue, err := dataEntry.DataValue.MarshalBinaryBase64()
	if err != nil {
		return details, err
	}

	details["data_value"] = dataValue

	return details, nil
}

func ClaimableBalanceDetails(claimableBalanceEntry *xdr.ClaimableBalanceEntry) (map[string]interface{}, error) {
	details := map[string]interface{}{}

	details["amount"] = int64(claimableBalanceEntry.Amount)
	details["flags"] = uint32(claimableBalanceEntry.Flags())

	var err error
	var balanceId string
	balanceId, err = claimableBalanceEntry.BalanceId.MarshalBinaryBase64()
	if err != nil {
		return details, err
	}

	details["balance_id"] = balanceId

	var assetType, assetCode, assetIssuer string
	err = claimableBalanceEntry.Asset.Extract(&assetType, &assetCode, &assetIssuer)
	if err != nil {
		return details, err
	}

	details["asset_code"] = assetCode
	details["asset_issuer"] = assetIssuer
	details["asset_type"] = assetType

	var claimants []Claimant
	for _, c := range claimableBalanceEntry.Claimants {
		switch c.Type {
		case 0:
			claimants = append(claimants, Claimant{
				Destination: c.V0.Destination.Address(),
				Predicate:   c.V0.Predicate,
			})
		}
	}

	details["claimants"] = claimants

	return details, nil
}

func LiquidityPoolDetails(liquidityPoolEntry *xdr.LiquidityPoolEntry) (map[string]interface{}, error) {
	details := map[string]interface{}{}

	details["liquidity_pool_type"] = liquidityPoolEntry.Body.Type

	var err error
	var liquidtiyPoolID string
	liquidtiyPoolID, err = liquidityPoolEntry.LiquidityPoolId.MarshalBinaryBase64()
	if err != nil {
		return details, err
	}

	details["liquidity_pool_id"] = liquidtiyPoolID

	var ok bool
	var constantProduct xdr.LiquidityPoolEntryConstantProduct
	constantProduct, ok = liquidityPoolEntry.Body.GetConstantProduct()
	if !ok {
		details["liquidity_pool_fee"] = int32(0)
		details["trustline_count"] = int64(0)
		details["pool_share_count"] = int64(0)
		details["asset_a_code"] = ""
		details["asset_a_issuer"] = ""
		details["asset_a_type"] = ""
		details["asset_a_reserve"] = int64(0)
		details["asset_b_code"] = ""
		details["asset_b_issuer"] = ""
		details["asset_b_type"] = ""
		details["asset_b_reserve"] = int64(0)
	} else {
		details["liquidity_pool_fee"] = int32(constantProduct.Params.Fee)
		details["trustline_count"] = int64(constantProduct.PoolSharesTrustLineCount)
		details["pool_share_count"] = int64(constantProduct.PoolSharesTrustLineCount)
		var assetAType, assetACode, assetAIssuer string
		err = constantProduct.Params.AssetA.Extract(&assetAType, &assetACode, &assetAIssuer)
		if err != nil {
			return details, err
		}

		details["asset_a_code"] = assetACode
		details["asset_a_issuer"] = assetAIssuer
		details["asset_a_type"] = assetAType
		details["asset_a_reserve"] = int64(constantProduct.ReserveA)

		var assetBType, assetBCode, assetBIssuer string
		err = constantProduct.Params.AssetB.Extract(&assetBType, &assetBCode, &assetBIssuer)
		if err != nil {
			return details, err
		}

		details["asset_b_code"] = assetBCode
		details["asset_b_issuer"] = assetBIssuer
		details["asset_b_type"] = assetBType
		details["asset_b_reserve"] = int64(constantProduct.ReserveB)
	}

	return details, nil
}

func ContractDataDetails(passphrase string, contractDataEntry *xdr.ContractDataEntry) (map[string]interface{}, error) {
	details := map[string]interface{}{}

	// This should use xdr2json
	details["contract_key"] = contractDataEntry.Key
	details["contract_val"] = contractDataEntry.Val

	details["contract_durability"] = contractDataEntry.Durability.String()

	contractAsset := AssetFromContractData(passphrase, *contractDataEntry)

	var err error
	var ok bool
	var assetType, assetCode, assetIssuer string
	err = contractAsset.Extract(&assetType, &assetCode, &assetIssuer)
	if err != nil {
		return details, err
	}

	details["asset_code"] = assetCode
	details["asset_issuer"] = assetIssuer
	details["asset_type"] = assetType

	var contractDataBalanceHolder string
	dataBalanceHolder, dataBalance, ok := ContractBalanceFromContractData(passphrase, *contractDataEntry)
	if ok {
		holderHashByte, _ := xdr.Hash(dataBalanceHolder).MarshalBinary()
		contractDataBalanceHolder, _ = strkey.Encode(strkey.VersionByteContract, holderHashByte)
		details["balance_holder"] = contractDataBalanceHolder
		details["balance"] = dataBalance.String()
	}

	var contractDataContractId xdr.Hash
	contractDataContractId, ok = contractDataEntry.Contract.GetContractId()
	if ok {
		var contractDataContractIdByte []byte
		contractDataContractIdByte, err = contractDataContractId.MarshalBinary()
		if err != nil {
			return details, err
		}

		var contractDataContractIdString string
		contractDataContractIdString, err = strkey.Encode(strkey.VersionByteContract, contractDataContractIdByte)
		if err != nil {
			return details, err
		}

		details["contract_id"] = contractDataContractIdString
	}

	return details, nil
}

func ContractCodeDetails(contractCodeEntry *xdr.ContractCodeEntry) (map[string]interface{}, error) {
	details := map[string]interface{}{}

	switch contractCodeEntry.Ext.V {
	case 1:
		details["n_instructions"] = uint32(contractCodeEntry.Ext.V1.CostInputs.NInstructions)
		details["n_functions"] = uint32(contractCodeEntry.Ext.V1.CostInputs.NFunctions)
		details["n_globals"] = uint32(contractCodeEntry.Ext.V1.CostInputs.NGlobals)
		details["n_table_entries"] = uint32(contractCodeEntry.Ext.V1.CostInputs.NTableEntries)
		details["n_types"] = uint32(contractCodeEntry.Ext.V1.CostInputs.NTypes)
		details["n_data_segments"] = uint32(contractCodeEntry.Ext.V1.CostInputs.NDataSegments)
		details["n_elem_segments"] = uint32(contractCodeEntry.Ext.V1.CostInputs.NElemSegments)
		details["n_imports"] = uint32(contractCodeEntry.Ext.V1.CostInputs.NImports)
		details["n_exports"] = uint32(contractCodeEntry.Ext.V1.CostInputs.NExports)
		details["n_data_segment_bytes"] = uint32(contractCodeEntry.Ext.V1.CostInputs.NDataSegmentBytes)
	default:
		return details, fmt.Errorf("unknown ContractCodeEntry.Ext.V")
	}

	var err error
	var contractCode string
	contractCode, err = contractCodeEntry.Hash.MarshalBinaryBase64()
	if err != nil {
		return details, err
	}

	details["contract_code_hash"] = contractCode

	return details, nil
}

func TtlDetails(ttlEntry *xdr.TtlEntry) (map[string]interface{}, error) {
	details := map[string]interface{}{}

	details["live_until_ledger_seq"] = uint32(ttlEntry.LiveUntilLedgerSeq)

	return details, nil
}

var (
	// these are storage DataKey enum
	// https://github.com/stellar/rs-soroban-env/blob/v0.0.16/soroban-env-host/src/native_contract/token/storage_types.rs#L23
	balanceMetadataSym = xdr.ScSymbol("Balance")
	issuerSym          = xdr.ScSymbol("issuer")
	assetCodeSym       = xdr.ScSymbol("asset_code")
	assetInfoSym       = xdr.ScSymbol("AssetInfo")
	assetInfoVec       = &xdr.ScVec{
		xdr.ScVal{
			Type: xdr.ScValTypeScvSymbol,
			Sym:  &assetInfoSym,
		},
	}
	assetInfoKey = xdr.ScVal{
		Type: xdr.ScValTypeScvVec,
		Vec:  &assetInfoVec,
	}
)

func AssetFromContractData(passphrase string, contractData xdr.ContractDataEntry) *xdr.Asset {
	if contractData.Key.Type != xdr.ScValTypeScvLedgerKeyContractInstance {
		return nil
	}
	contractInstanceData, ok := contractData.Val.GetInstance()
	if !ok || contractInstanceData.Storage == nil {
		return nil
	}

	nativeAssetContractID, err := xdr.MustNewNativeAsset().ContractID(passphrase)
	if err != nil {
		return nil
	}

	var assetInfo *xdr.ScVal
	for _, mapEntry := range *contractInstanceData.Storage {
		if mapEntry.Key.Equals(assetInfoKey) {
			// clone the map entry to avoid reference to loop iterator
			mapValXdr, cloneErr := mapEntry.Val.MarshalBinary()
			if cloneErr != nil {
				return nil
			}
			assetInfo = &xdr.ScVal{}
			cloneErr = assetInfo.UnmarshalBinary(mapValXdr)
			if cloneErr != nil {
				return nil
			}
			break
		}
	}

	if assetInfo == nil {
		return nil
	}

	vecPtr, ok := assetInfo.GetVec()
	if !ok || vecPtr == nil || len(*vecPtr) != 2 {
		return nil
	}
	vec := *vecPtr

	sym, ok := vec[0].GetSym()
	if !ok {
		return nil
	}
	switch sym {
	case "AlphaNum4":
	case "AlphaNum12":
	case "Native":
		if contractData.Contract.ContractId != nil && (*contractData.Contract.ContractId) == nativeAssetContractID {
			asset := xdr.MustNewNativeAsset()
			return &asset
		}
	default:
		return nil
	}

	var assetCode, assetIssuer string
	assetMapPtr, ok := vec[1].GetMap()
	if !ok || assetMapPtr == nil || len(*assetMapPtr) != 2 {
		return nil
	}
	assetMap := *assetMapPtr

	assetCodeEntry, assetIssuerEntry := assetMap[0], assetMap[1]
	if sym, ok = assetCodeEntry.Key.GetSym(); !ok || sym != assetCodeSym {
		return nil
	}
	assetCodeSc, ok := assetCodeEntry.Val.GetStr()
	if !ok {
		return nil
	}
	if assetCode = string(assetCodeSc); assetCode == "" {
		return nil
	}

	if sym, ok = assetIssuerEntry.Key.GetSym(); !ok || sym != issuerSym {
		return nil
	}
	assetIssuerSc, ok := assetIssuerEntry.Val.GetBytes()
	if !ok {
		return nil
	}
	assetIssuer, err = strkey.Encode(strkey.VersionByteAccountID, assetIssuerSc)
	if err != nil {
		return nil
	}

	asset, err := xdr.NewCreditAsset(assetCode, assetIssuer)
	if err != nil {
		return nil
	}

	expectedID, err := asset.ContractID(passphrase)
	if err != nil {
		return nil
	}
	if contractData.Contract.ContractId == nil || expectedID != *(contractData.Contract.ContractId) {
		return nil
	}

	return &asset
}

// ContractBalanceFromContractData takes a ledger entry and verifies that the
// ledger entry corresponds to the balance entry written to contract storage by
// the Stellar Asset Contract.
//
// Reference:
//
//	https://github.com/stellar/rs-soroban-env/blob/da325551829d31dcbfa71427d51c18e71a121c5f/soroban-env-host/src/native_contract/token/storage_types.rs#L11-L24
func ContractBalanceFromContractData(passphrase string, contractData xdr.ContractDataEntry) ([32]byte, *big.Int, bool) {
	_, err := xdr.MustNewNativeAsset().ContractID(passphrase)
	if err != nil {
		return [32]byte{}, nil, false
	}

	if contractData.Contract.ContractId == nil {
		return [32]byte{}, nil, false
	}

	keyEnumVecPtr, ok := contractData.Key.GetVec()
	if !ok || keyEnumVecPtr == nil {
		return [32]byte{}, nil, false
	}
	keyEnumVec := *keyEnumVecPtr
	if len(keyEnumVec) != 2 || !keyEnumVec[0].Equals(
		xdr.ScVal{
			Type: xdr.ScValTypeScvSymbol,
			Sym:  &balanceMetadataSym,
		},
	) {
		return [32]byte{}, nil, false
	}

	scAddress, ok := keyEnumVec[1].GetAddress()
	if !ok {
		return [32]byte{}, nil, false
	}

	holder, ok := scAddress.GetContractId()
	if !ok {
		return [32]byte{}, nil, false
	}

	balanceMapPtr, ok := contractData.Val.GetMap()
	if !ok || balanceMapPtr == nil {
		return [32]byte{}, nil, false
	}
	balanceMap := *balanceMapPtr
	if !ok || len(balanceMap) != 3 {
		return [32]byte{}, nil, false
	}

	var keySym xdr.ScSymbol
	if keySym, ok = balanceMap[0].Key.GetSym(); !ok || keySym != "amount" {
		return [32]byte{}, nil, false
	}
	if keySym, ok = balanceMap[1].Key.GetSym(); !ok || keySym != "authorized" ||
		!balanceMap[1].Val.IsBool() {
		return [32]byte{}, nil, false
	}
	if keySym, ok = balanceMap[2].Key.GetSym(); !ok || keySym != "clawback" ||
		!balanceMap[2].Val.IsBool() {
		return [32]byte{}, nil, false
	}
	amount, ok := balanceMap[0].Val.GetI128()
	if !ok {
		return [32]byte{}, nil, false
	}

	// amount cannot be negative
	// https://github.com/stellar/rs-soroban-env/blob/a66f0815ba06a2f5328ac420950690fd1642f887/soroban-env-host/src/native_contract/token/balance.rs#L92-L93
	if int64(amount.Hi) < 0 {
		return [32]byte{}, nil, false
	}
	amt := new(big.Int).Lsh(new(big.Int).SetInt64(int64(amount.Hi)), 64)
	amt.Add(amt, new(big.Int).SetUint64(uint64(amount.Lo)))
	return holder, amt, true
}
