package main

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	"github.com/stellar/go/hash"
	"github.com/stellar/go/ingest"
	"github.com/stellar/go/strkey"
	"github.com/stellar/go/xdr"
)

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

type AssetFromContractDataFunc func(ledgerEntry xdr.LedgerEntry, passphrase string) *xdr.Asset
type ContractBalanceFromContractDataFunc func(ledgerEntry xdr.LedgerEntry, passphrase string) ([32]byte, *big.Int, bool)

type TransformContractDataStruct struct {
	AssetFromContractData           AssetFromContractDataFunc
	ContractBalanceFromContractData ContractBalanceFromContractDataFunc
}

func NewTransformContractDataStruct(assetFrom AssetFromContractDataFunc, contractBalance ContractBalanceFromContractDataFunc) *TransformContractDataStruct {
	return &TransformContractDataStruct{
		AssetFromContractData:           assetFrom,
		ContractBalanceFromContractData: contractBalance,
	}
}

type ContractDataOutput struct {
	ContractId                string `json:"contract_id"`
	ContractKeyType           string `json:"contract_key_type"`
	ContractDurability        string `json:"contract_durability"`
	ContractDataAssetCode     string `json:"asset_code"`
	ContractDataAssetIssuer   string `json:"asset_issuer"`
	ContractDataAssetType     string `json:"asset_type"`
	ContractDataBalanceHolder string `json:"balance_holder"`
	ContractDataBalance       string `json:"balance"` // balance is a string because it is go type big.Int
	LedgerSequence            uint32 `json:"ledger_sequence"`
	LedgerKeyHash             string `json:"ledger_key_hash"`
}

func (t *TransformContractDataStruct) TransformContractData(ledgerChange ingest.Change, passphrase string, header xdr.LedgerHeaderHistoryEntry) (ContractDataOutput, error, bool) {
	ledgerEntry, _, _, err := ExtractEntryFromChange(ledgerChange)
	if err != nil {
		return ContractDataOutput{}, err, false
	}

	contractData, ok := ledgerEntry.Data.GetContractData()
	if !ok {
		return ContractDataOutput{}, fmt.Errorf("could not extract contract data from ledger entry; actual type is %s", ledgerEntry.Data.Type), false
	}

	if contractData.Key.Type.String() == "ScValTypeScvLedgerKeyNonce" {
		// Is a nonce and should be discarded
		return ContractDataOutput{}, nil, false
	}

	ledgerKeyHash := LedgerEntryToLedgerKeyHash(ledgerEntry)

	var contractDataAssetType string
	var contractDataAssetCode string
	var contractDataAssetIssuer string

	contractDataAsset := t.AssetFromContractData(ledgerEntry, passphrase)
	if contractDataAsset != nil {
		contractDataAssetType = contractDataAsset.Type.String()
		contractDataAssetCode = contractDataAsset.GetCode()
		contractDataAssetCode = strings.ReplaceAll(contractDataAssetCode, "\x00", "")
		contractDataAssetIssuer = contractDataAsset.GetIssuer()
	}

	var contractDataBalanceHolder string
	var contractDataBalance string

	dataBalanceHolder, dataBalance, _ := t.ContractBalanceFromContractData(ledgerEntry, passphrase)
	if dataBalance != nil {
		holderHashByte, _ := xdr.Hash(dataBalanceHolder).MarshalBinary()
		contractDataBalanceHolder, _ = strkey.Encode(strkey.VersionByteContract, holderHashByte)
		contractDataBalance = dataBalance.String()
	}

	contractDataContractId, ok := contractData.Contract.GetContractId()
	if !ok {
		return ContractDataOutput{}, fmt.Errorf("could not extract contractId data information from contractData"), false
	}

	contractDataKeyType := contractData.Key.Type.String()
	contractDataContractIdByte, _ := contractDataContractId.MarshalBinary()
	outputContractDataContractId, _ := strkey.Encode(strkey.VersionByteContract, contractDataContractIdByte)

	contractDataDurability := contractData.Durability.String()

	ledgerSequence := header.Header.LedgerSeq

	transformedData := ContractDataOutput{
		ContractId:                outputContractDataContractId,
		ContractKeyType:           contractDataKeyType,
		ContractDurability:        contractDataDurability,
		ContractDataAssetCode:     contractDataAssetCode,
		ContractDataAssetIssuer:   contractDataAssetIssuer,
		ContractDataAssetType:     contractDataAssetType,
		ContractDataBalanceHolder: contractDataBalanceHolder,
		ContractDataBalance:       contractDataBalance,
		LedgerSequence:            uint32(ledgerSequence),
		LedgerKeyHash:             ledgerKeyHash,
	}
	return transformedData, nil, true
}

type ContractCodeOutput struct {
	ContractCodeHash string `json:"contract_code_hash"`
	LedgerKeyHash    string `json:"ledger_key_hash"`
	LedgerSequence   uint32 `json:"ledger_sequence"`
}

func AssetFromContractData(ledgerEntry xdr.LedgerEntry, passphrase string) *xdr.Asset {
	contractData, ok := ledgerEntry.Data.GetContractData()
	if !ok {
		return nil
	}
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
func ContractBalanceFromContractData(ledgerEntry xdr.LedgerEntry, passphrase string) ([32]byte, *big.Int, bool) {
	contractData, ok := ledgerEntry.Data.GetContractData()
	if !ok {
		return [32]byte{}, nil, false
	}

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

func TransformContractCode(ledgerChange ingest.Change, header xdr.LedgerHeaderHistoryEntry) ContractCodeOutput {
	ledgerEntry, _, _, err := ExtractEntryFromChange(ledgerChange)
	if err != nil {
		return ContractCodeOutput{}
	}

	contractCode, ok := ledgerEntry.Data.GetContractCode()
	if !ok {
		return ContractCodeOutput{}
	}

	// LedgerEntryChange must contain a contract code change to be parsed, otherwise skip
	if ledgerEntry.Data.Type != xdr.LedgerEntryTypeContractCode {
		return ContractCodeOutput{}
	}

	ledgerKeyHash := LedgerEntryToLedgerKeyHash(ledgerEntry)
	contractCodeHash := contractCode.Hash.HexString()
	ledgerSequence := header.Header.LedgerSeq

	return ContractCodeOutput{
		ContractCodeHash: contractCodeHash,
		LedgerKeyHash:    ledgerKeyHash,
		LedgerSequence:   uint32(ledgerSequence),
	}

}

func LedgerEntryToLedgerKeyHash(ledgerEntry xdr.LedgerEntry) string {
	ledgerKey, _ := ledgerEntry.LedgerKey()
	ledgerKeyByte, _ := ledgerKey.MarshalBinary()
	hashedLedgerKeyByte := hash.Hash(ledgerKeyByte)
	ledgerKeyHash := hex.EncodeToString(hashedLedgerKeyByte[:])

	return ledgerKeyHash
}

type ttlOutput struct {
	KeyHash            string `json:"key_hash"` // key_hash is contract_code_hash or contract_id
	LiveUntilLedgerSeq uint32 `json:"live_until_ledger_seq"`
	LedgerSequence     uint32 `json:"ledger_sequence"`
}

func TransformTtl(ledgerChange ingest.Change, header xdr.LedgerHeaderHistoryEntry) ttlOutput {
	ledgerEntry, _, _, err := ExtractEntryFromChange(ledgerChange)
	if err != nil {
		return ttlOutput{}
	}

	ttl, ok := ledgerEntry.Data.GetTtl()
	if !ok {
		return ttlOutput{}
	}

	// LedgerEntryChange must contain a ttl change to be parsed, otherwise skip
	if ledgerEntry.Data.Type != xdr.LedgerEntryTypeTtl {
		return ttlOutput{}
	}

	keyHash := ttl.KeyHash.HexString()
	liveUntilLedgerSeq := ttl.LiveUntilLedgerSeq
	ledgerSequence := header.Header.LedgerSeq

	return ttlOutput{
		KeyHash:            keyHash,
		LiveUntilLedgerSeq: uint32(liveUntilLedgerSeq),
		LedgerSequence:     uint32(ledgerSequence),
	}
}

// ExtractEntryFromChange gets the most recent state of an entry from an ingestio change, as well as if the entry was deleted
func ExtractEntryFromChange(change ingest.Change) (xdr.LedgerEntry, xdr.LedgerEntryChangeType, bool, error) {
	switch changeType := change.LedgerEntryChangeType(); changeType {
	case xdr.LedgerEntryChangeTypeLedgerEntryCreated, xdr.LedgerEntryChangeTypeLedgerEntryUpdated:
		return *change.Post, changeType, false, nil
	case xdr.LedgerEntryChangeTypeLedgerEntryRemoved:
		return *change.Pre, changeType, true, nil
	default:
		return xdr.LedgerEntry{}, changeType, false, fmt.Errorf("unable to extract ledger entry type from change")
	}
}
