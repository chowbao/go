package xdrill

import (
	"encoding/base64"
	"math/big"
	"strings"
	"time"

	"github.com/stellar/go/strkey"
	"github.com/stellar/go/xdr"
)

type ContractDataEntry struct {
	contractDataEntry *xdr.ContractDataEntry
	change            *Change
	networkPassphrase string
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

func (c ContractDataEntry) ContractID() string {
	contractDataContractId := c.contractDataEntry.Contract.MustContractId()
	contractDataContractIdByte, err := contractDataContractId.MarshalBinary()
	if err != nil {
		panic(err)
	}

	contractDataContractIdString, err := strkey.Encode(strkey.VersionByteContract, contractDataContractIdByte)
	if err != nil {
		panic(err)
	}

	return contractDataContractIdString
}

func (c ContractDataEntry) ContractKeyType() string {
	return c.contractDataEntry.Key.Type.String()
}

func (c ContractDataEntry) ContractDurability() string {
	return c.contractDataEntry.Durability.String()
}

func (c ContractDataEntry) AssetCode() string {
	asset := c.AssetFromContractData(c.networkPassphrase)
	assetCode := asset.GetCode()

	return strings.ReplaceAll(assetCode, "\x00", "")
}

func (c ContractDataEntry) AssetIssuer() string {
	asset := c.AssetFromContractData(c.networkPassphrase)

	return asset.GetIssuer()
}

func (c ContractDataEntry) AssetType() string {
	asset := c.AssetFromContractData(c.networkPassphrase)

	return asset.Type.String()
}

func (c ContractDataEntry) BalanceHolder() string {
	var contractDataBalanceHolder string

	dataBalanceHolder, dataBalance, _ := c.ContractBalanceFromContractData(c.networkPassphrase)
	if dataBalance != nil {
		holderHashByte, _ := xdr.Hash(dataBalanceHolder).MarshalBinary()
		contractDataBalanceHolder, _ = strkey.Encode(strkey.VersionByteContract, holderHashByte)
	}

	return contractDataBalanceHolder
}

func (c ContractDataEntry) Balance() string {
	var contractDataBalance string

	_, dataBalance, _ := c.ContractBalanceFromContractData(c.networkPassphrase)
	if dataBalance != nil {
		contractDataBalance = dataBalance.String()
	}

	return contractDataBalance
}

func (c ContractDataEntry) LedgerKeyHash() string {
	return c.change.LedgerKeyHash()
}

func (c ContractDataEntry) Key() map[string]string {
	_, key := serializeScVal(c.contractDataEntry.Key)
	return key
}

func (c ContractDataEntry) Val() map[string]string {
	_, val := serializeScVal(c.contractDataEntry.Val)
	return val
}

func (c ContractDataEntry) Sponsor() string {
	return c.change.Sponsor()
}

func (c ContractDataEntry) LastModifiedLedger() uint32 {
	return c.change.LastModifiedLedger()
}

func (c ContractDataEntry) LedgerEntryChangeType() uint32 {
	return c.change.Type()
}

func (c ContractDataEntry) Deleted() bool {
	return c.change.Deleted()
}

func (c ContractDataEntry) ClosedAt() time.Time {
	return c.change.ClosedAt()
}

func (c ContractDataEntry) Sequence() uint32 {
	return c.change.Sequence()
}

func (c ContractDataEntry) AssetFromContractData(passphrase string) *xdr.Asset {
	ledgerEntry, _, _, err := c.change.ExtractEntryFromChange()
	if err != nil {
		return nil
	}

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
func (c ContractDataEntry) ContractBalanceFromContractData(passphrase string) ([32]byte, *big.Int, bool) {
	ledgerEntry, _, _, err := c.change.ExtractEntryFromChange()
	if err != nil {
		return [32]byte{}, nil, false
	}

	contractData, ok := ledgerEntry.Data.GetContractData()
	if !ok {
		return [32]byte{}, nil, false
	}

	_, err = xdr.MustNewNativeAsset().ContractID(passphrase)
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

func serializeScVal(scVal xdr.ScVal) (map[string]string, map[string]string) {
	serializedData := map[string]string{}
	serializedData["value"] = "n/a"
	serializedData["type"] = "n/a"

	serializedDataDecoded := map[string]string{}
	serializedDataDecoded["value"] = "n/a"
	serializedDataDecoded["type"] = "n/a"

	if scValTypeName, ok := scVal.ArmForSwitch(int32(scVal.Type)); ok {
		serializedData["type"] = scValTypeName
		serializedDataDecoded["type"] = scValTypeName
		if raw, err := scVal.MarshalBinary(); err == nil {
			serializedData["value"] = base64.StdEncoding.EncodeToString(raw)
			serializedDataDecoded["value"] = scVal.String()
		}
	}

	return serializedData, serializedDataDecoded
}
