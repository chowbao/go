package utils

import (
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/dgryski/go-farm"
	"github.com/stellar/go/hash"
	"github.com/stellar/go/strkey"
	"github.com/stellar/go/xdr"
)

// HashToHexString is utility function that converts and xdr.Hash type to a hex string
func HashToHexString(inputHash xdr.Hash) string {
	sliceHash := inputHash[:]
	hexString := hex.EncodeToString(sliceHash)
	return hexString
}

type ID struct {
	LedgerSequence   int32
	TransactionOrder int32
	OperationOrder   int32
}

const (
	// LedgerMask is the bitmask to mask out ledger sequences in a
	// TotalOrderID
	LedgerMask = (1 << 32) - 1
	// TransactionMask is the bitmask to mask out transaction indexes
	TransactionMask = (1 << 20) - 1
	// OperationMask is the bitmask to mask out operation indexes
	OperationMask = (1 << 12) - 1

	// LedgerShift is the number of bits to shift an int64 to target the
	// ledger component
	LedgerShift = 32
	// TransactionShift is the number of bits to shift an int64 to
	// target the transaction component
	TransactionShift = 12
	// OperationShift is the number of bits to shift an int64 to target
	// the operation component
	OperationShift = 0
)

// New creates a new total order ID
func NewID(ledger int32, tx int32, op int32) *ID {
	return &ID{
		LedgerSequence:   ledger,
		TransactionOrder: tx,
		OperationOrder:   op,
	}
}

// ToInt64 converts this struct back into an int64
func (id ID) ToInt64() (result int64) {

	if id.LedgerSequence < 0 {
		panic("invalid ledger sequence")
	}

	if id.TransactionOrder > TransactionMask {
		panic("transaction order overflow")
	}

	if id.OperationOrder > OperationMask {
		panic("operation order overflow")
	}

	result = result | ((int64(id.LedgerSequence) & LedgerMask) << LedgerShift)
	result = result | ((int64(id.TransactionOrder) & TransactionMask) << TransactionShift)
	result = result | ((int64(id.OperationOrder) & OperationMask) << OperationShift)
	return
}

// TODO: This should be moved into the go monorepo xdr functions
// Or nodeID should just be an xdr.AccountId but the error message would be incorrect
func GetAddress(nodeID xdr.NodeId) (string, bool) {
	switch nodeID.Type {
	case xdr.PublicKeyTypePublicKeyTypeEd25519:
		ed, ok := nodeID.GetEd25519()
		if !ok {
			return "", false
		}
		raw := make([]byte, 32)
		copy(raw, ed[:])
		encodedAddress, err := strkey.Encode(strkey.VersionByteAccountID, raw)
		if err != nil {
			return "", false
		}
		return encodedAddress, true
	default:
		return "", false
	}
}

// GetAccountAddressFromMuxedAccount takes in a muxed account and returns the address of the account
func GetAccountAddressFromMuxedAccount(account xdr.MuxedAccount) (string, error) {
	providedID := account.ToAccountId()
	pointerToID := &providedID
	return pointerToID.GetAddress()
}

// TimePointToUTCTimeStamp takes in an xdr TimePoint and converts it to a time.Time struct in UTC. It returns an error for negative timepoints
func TimePointToUTCTimeStamp(providedTime xdr.TimePoint) (time.Time, error) {
	intTime := int64(providedTime)
	if intTime < 0 {
		return time.Now(), errors.New("the timepoint is negative")
	}
	return time.Unix(intTime, 0).UTC(), nil
}

func GetAccountBalanceFromLedgerEntryChanges(changes xdr.LedgerEntryChanges, sourceAccountAddress string) (int64, int64) {
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

func GetTxSigners(xdrSignatures []xdr.DecoratedSignature) ([]string, error) {
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

// Path is a representation of an asset without an ID that forms part of a path in a path payment
type Path struct {
	AssetCode   string `json:"asset_code"`
	AssetIssuer string `json:"asset_issuer"`
	AssetType   string `json:"asset_type"`
}

func TransformPath(initialPath []xdr.Asset) []Path {
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

func ConvertStroopValueToReal(input xdr.Int64) float64 {
	output, _ := big.NewRat(int64(input), int64(10000000)).Float64()
	return output
}

func FarmHashAsset(assetCode, assetIssuer, assetType string) int64 {
	asset := fmt.Sprintf("%s%s%s", assetCode, assetIssuer, assetType)
	hash := farm.Fingerprint64([]byte(asset))

	return int64(hash)
}

func FormatPrefix(p string) string {
	if p != "" {
		p += "_"
	}
	return p
}

type Price struct {
	Numerator   int32 `json:"n"`
	Denominator int32 `json:"d"`
}

func PoolIDToString(id xdr.PoolId) string {
	return xdr.Hash(id).HexString()
}

type Claimant struct {
	Destination string             `json:"destination"`
	Predicate   xdr.ClaimPredicate `json:"predicate"`
}

func TransformClaimants(claimants []xdr.Claimant) []Claimant {
	var transformed []Claimant
	for _, c := range claimants {
		switch c.Type {
		case 0:
			transformed = append(transformed, Claimant{
				Destination: c.V0.Destination.Address(),
				Predicate:   c.V0.Predicate,
			})
		}
	}
	return transformed
}

type SponsorshipOutput struct {
	Operation      xdr.Operation
	OperationIndex uint32
}

type LiquidityPoolDelta struct {
	ReserveA        xdr.Int64
	ReserveB        xdr.Int64
	TotalPoolShares xdr.Int64
}

func LedgerKeyToLedgerKeyHash(ledgerKey xdr.LedgerKey) string {
	ledgerKeyByte, _ := ledgerKey.MarshalBinary()
	hashedLedgerKeyByte := hash.Hash(ledgerKeyByte)
	ledgerKeyHash := hex.EncodeToString(hashedLedgerKeyByte[:])

	return ledgerKeyHash
}

type Asset struct {
	AssetCode   string
	AssetIssuer string
	AssetType   string
	AssetID     int64
}

func TransformSingleAsset(asset xdr.Asset) (Asset, error) {
	var outputAssetType, outputAssetCode, outputAssetIssuer string
	err := asset.Extract(&outputAssetType, &outputAssetCode, &outputAssetIssuer)
	if err != nil {
		return Asset{}, fmt.Errorf("could not extract asset from this operation")
	}

	farmAssetID := FarmHashAsset(outputAssetCode, outputAssetIssuer, outputAssetType)

	return Asset{
		AssetCode:   outputAssetCode,
		AssetIssuer: outputAssetIssuer,
		AssetType:   outputAssetType,
		AssetID:     farmAssetID,
	}, nil
}

func TransformTrustLineAsset(asset xdr.TrustLineAsset) (Asset, error) {
	var outputAssetType, outputAssetCode, outputAssetIssuer string
	err := asset.Extract(&outputAssetType, &outputAssetCode, &outputAssetIssuer)
	if err != nil {
		return Asset{}, fmt.Errorf("could not extract asset from this operation")
	}

	farmAssetID := FarmHashAsset(outputAssetCode, outputAssetIssuer, outputAssetType)

	return Asset{
		AssetCode:   outputAssetCode,
		AssetIssuer: outputAssetIssuer,
		AssetType:   outputAssetType,
		AssetID:     farmAssetID,
	}, nil
}

func LedgerEntryToLedgerKeyHash(ledgerEntry xdr.LedgerEntry) string {
	ledgerKey, _ := ledgerEntry.LedgerKey()
	ledgerKeyByte, _ := ledgerKey.MarshalBinary()
	hashedLedgerKeyByte := hash.Hash(ledgerKeyByte)
	ledgerKeyHash := hex.EncodeToString(hashedLedgerKeyByte[:])

	return ledgerKeyHash
}

func SerializeScVal(scVal xdr.ScVal) (map[string]string, map[string]string) {
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
