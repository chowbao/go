package ledgerentry

import (
	"math/big"

	"github.com/stellar/go/strkey"
	"github.com/stellar/go/xdr"
)

type ContractData struct {
	ContractId      string      `json:"contract_id"`
	ContractKeyType string      `json:"contract_key_type"`
	Durability      string      `json:"durability"`
	AssetCode       string      `json:"asset_code"`
	AssetIssuer     string      `json:"asset_issuer"`
	AssetType       string      `json:"asset_type"`
	BalanceHolder   string      `json:"balance_holder"`
	Balance         string      `json:"balance"`
	KeyDecoded      interface{} `json:"key_decoded"`
	ValDecoded      interface{} `json:"val_decoded"`
}

func ContractDataDetails(passphrase string, contractDataEntry *xdr.ContractDataEntry) (ContractData, error) {
	contractData := ContractData{
		Durability: contractDataEntry.Durability.String(),
		KeyDecoded: contractDataEntry.Key,
		ValDecoded: contractDataEntry.Val,
	}

	contractAsset := AssetFromContractData(passphrase, *contractDataEntry)

	var err error
	var ok bool
	var assetType, assetCode, assetIssuer string
	err = contractAsset.Extract(&assetType, &assetCode, &assetIssuer)
	if err != nil {
		return ContractData{}, err
	}

	contractData.AssetCode = assetCode
	contractData.AssetIssuer = assetIssuer
	contractData.AssetType = assetType

	var contractDataBalanceHolder string
	dataBalanceHolder, dataBalance, ok := ContractBalanceFromContractData(passphrase, *contractDataEntry)
	if ok {
		holderHashByte, _ := xdr.Hash(dataBalanceHolder).MarshalBinary()
		contractDataBalanceHolder, _ = strkey.Encode(strkey.VersionByteContract, holderHashByte)
		contractData.BalanceHolder = contractDataBalanceHolder
		contractData.Balance = dataBalance.String()
	}

	var contractDataContractId xdr.Hash
	contractDataContractId, ok = contractDataEntry.Contract.GetContractId()
	if ok {
		var contractDataContractIdByte []byte
		contractDataContractIdByte, err = contractDataContractId.MarshalBinary()
		if err != nil {
			return ContractData{}, err
		}

		var contractDataContractIdString string
		contractDataContractIdString, err = strkey.Encode(strkey.VersionByteContract, contractDataContractIdByte)
		if err != nil {
			return ContractData{}, err
		}

		contractData.ContractId = contractDataContractIdString
	}

	return contractData, nil
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
