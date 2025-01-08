package xdrill

import (
	"strconv"
	"time"

	"github.com/stellar/go/xdr"
)

type ConfigSettingEntry struct {
	configSettingEntry *xdr.ConfigSettingEntry
	change             *Change
}

func (c ConfigSettingEntry) ConfigSettingID() int32 {
	return int32(c.configSettingEntry.ConfigSettingId)
}

func (c ConfigSettingEntry) ContractMaxSizeBytes() uint32 {
	bytes, ok := c.configSettingEntry.GetContractMaxSizeBytes()
	if !ok {
		return 0
	}

	return uint32(bytes)
}

func (c ConfigSettingEntry) LedgerMaxInstructions() int64 {
	contractCompute, ok := c.configSettingEntry.GetContractCompute()
	if !ok {
		return 0
	}

	return int64(contractCompute.LedgerMaxInstructions)
}

func (c ConfigSettingEntry) TxMaxInstructions() int64 {
	contractCompute, ok := c.configSettingEntry.GetContractCompute()
	if !ok {
		return 0
	}

	return int64(contractCompute.TxMaxInstructions)
}

func (c ConfigSettingEntry) FeeRatePerInstructionsIncrement() int64 {
	contractCompute, ok := c.configSettingEntry.GetContractCompute()
	if !ok {
		return 0
	}

	return int64(contractCompute.FeeRatePerInstructionsIncrement)
}

func (c ConfigSettingEntry) TxMemoryLimit() uint32 {
	contractCompute, ok := c.configSettingEntry.GetContractCompute()
	if !ok {
		return 0
	}

	return uint32(contractCompute.TxMemoryLimit)
}

func (c ConfigSettingEntry) LedgerMaxReadLedgerEntries() uint32 {
	contractLedgerCost, ok := c.configSettingEntry.GetContractLedgerCost()
	if !ok {
		return 0
	}

	return uint32(contractLedgerCost.LedgerMaxReadLedgerEntries)
}

func (c ConfigSettingEntry) LedgerMaxReadBytes() uint32 {
	contractLedgerCost, ok := c.configSettingEntry.GetContractLedgerCost()
	if !ok {
		return 0
	}

	return uint32(contractLedgerCost.LedgerMaxReadBytes)
}

func (c ConfigSettingEntry) LedgerMaxWriteLedgerEntries() uint32 {
	contractLedgerCost, ok := c.configSettingEntry.GetContractLedgerCost()
	if !ok {
		return 0
	}

	return uint32(contractLedgerCost.LedgerMaxWriteLedgerEntries)
}

func (c ConfigSettingEntry) LedgerMaxWriteBytes() uint32 {
	contractLedgerCost, ok := c.configSettingEntry.GetContractLedgerCost()
	if !ok {
		return 0
	}

	return uint32(contractLedgerCost.LedgerMaxWriteBytes)
}

func (c ConfigSettingEntry) TxMaxReadLedgerEntries() uint32 {
	contractLedgerCost, ok := c.configSettingEntry.GetContractLedgerCost()
	if !ok {
		return 0
	}

	return uint32(contractLedgerCost.TxMaxReadLedgerEntries)
}

func (c ConfigSettingEntry) TxMaxReadBytes() uint32 {
	contractLedgerCost, ok := c.configSettingEntry.GetContractLedgerCost()
	if !ok {
		return 0
	}

	return uint32(contractLedgerCost.TxMaxReadBytes)
}

func (c ConfigSettingEntry) TxMaxWriteLedgerEntries() uint32 {
	contractLedgerCost, ok := c.configSettingEntry.GetContractLedgerCost()
	if !ok {
		return 0
	}

	return uint32(contractLedgerCost.TxMaxWriteLedgerEntries)
}

func (c ConfigSettingEntry) TxMaxWriteBytes() uint32 {
	contractLedgerCost, ok := c.configSettingEntry.GetContractLedgerCost()
	if !ok {
		return 0
	}

	return uint32(contractLedgerCost.TxMaxWriteBytes)
}

func (c ConfigSettingEntry) FeeReadLedgerEntry() int64 {
	contractLedgerCost, ok := c.configSettingEntry.GetContractLedgerCost()
	if !ok {
		return 0
	}

	return int64(contractLedgerCost.FeeReadLedgerEntry)
}

func (c ConfigSettingEntry) FeeWriteLedgerEntry() int64 {
	contractLedgerCost, ok := c.configSettingEntry.GetContractLedgerCost()
	if !ok {
		return 0
	}

	return int64(contractLedgerCost.FeeWriteLedgerEntry)
}

func (c ConfigSettingEntry) FeeRead1Kb() int64 {
	contractLedgerCost, ok := c.configSettingEntry.GetContractLedgerCost()
	if !ok {
		return 0
	}

	return int64(contractLedgerCost.FeeRead1Kb)
}

func (c ConfigSettingEntry) BucketListTargetSizeBytes() int64 {
	contractLedgerCost, ok := c.configSettingEntry.GetContractLedgerCost()
	if !ok {
		return 0
	}

	return int64(contractLedgerCost.BucketListTargetSizeBytes)
}

func (c ConfigSettingEntry) WriteFee1KbBucketListLow() int64 {
	contractLedgerCost, ok := c.configSettingEntry.GetContractLedgerCost()
	if !ok {
		return 0
	}

	return int64(contractLedgerCost.WriteFee1KbBucketListLow)
}

func (c ConfigSettingEntry) WriteFee1KbBucketListHigh() int64 {
	contractLedgerCost, ok := c.configSettingEntry.GetContractLedgerCost()
	if !ok {
		return 0
	}

	return int64(contractLedgerCost.WriteFee1KbBucketListHigh)
}

func (c ConfigSettingEntry) BucketListWriteFeeGrowthFactor() uint32 {
	contractLedgerCost, ok := c.configSettingEntry.GetContractLedgerCost()
	if !ok {
		return 0
	}

	return uint32(contractLedgerCost.BucketListWriteFeeGrowthFactor)
}

func (c ConfigSettingEntry) FeeHistorical1Kb() int64 {
	contractHistoricalData, ok := c.configSettingEntry.GetContractHistoricalData()
	if !ok {
		return 0
	}

	return int64(contractHistoricalData.FeeHistorical1Kb)
}

func (c ConfigSettingEntry) TxMaxContractEventsSizeBytes() uint32 {
	contractEvents, ok := c.configSettingEntry.GetContractEvents()
	if !ok {
		return 0
	}

	return uint32(contractEvents.TxMaxContractEventsSizeBytes)
}

func (c ConfigSettingEntry) FeeContractEvents1Kb() int64 {
	contractEvents, ok := c.configSettingEntry.GetContractEvents()
	if !ok {
		return 0
	}

	return int64(contractEvents.FeeContractEvents1Kb)
}

func (c ConfigSettingEntry) LedgerMaxTxsSizeBytes() uint32 {
	contractBandwidth, ok := c.configSettingEntry.GetContractBandwidth()
	if !ok {
		return 0
	}

	return uint32(contractBandwidth.LedgerMaxTxsSizeBytes)
}

func (c ConfigSettingEntry) TxMaxSizeBytes() uint32 {
	contractBandwidth, ok := c.configSettingEntry.GetContractBandwidth()
	if !ok {
		return 0
	}

	return uint32(contractBandwidth.TxMaxSizeBytes)
}

func (c ConfigSettingEntry) FeeTxSize1Kb() int64 {
	contractBandwidth, ok := c.configSettingEntry.GetContractBandwidth()
	if !ok {
		return 0
	}

	return int64(contractBandwidth.FeeTxSize1Kb)
}

func (c ConfigSettingEntry) ContractCostParamsCpuInsns() []map[string]string {
	contractCostParamsCpuInsns, ok := c.configSettingEntry.GetContractCostParamsCpuInsns()
	if !ok {
		return []map[string]string{}
	}

	return serializeCostParams(contractCostParamsCpuInsns)
}

func (c ConfigSettingEntry) ContractCostParamsMemBytes() []map[string]string {
	contractCostParamsMemBytes, ok := c.configSettingEntry.GetContractCostParamsMemBytes()
	if !ok {
		return []map[string]string{}
	}

	return serializeCostParams(contractCostParamsMemBytes)
}

func (c ConfigSettingEntry) ContractDataKeySizeBytes() uint32 {
	bytes, ok := c.configSettingEntry.GetContractDataKeySizeBytes()
	if !ok {
		return 0
	}

	return uint32(bytes)
}

func (c ConfigSettingEntry) ContractDataEntrySizeBytes() uint32 {
	bytes, ok := c.configSettingEntry.GetContractDataEntrySizeBytes()
	if !ok {
		return 0
	}

	return uint32(bytes)
}

func (c ConfigSettingEntry) MaxEntryTtl() uint32 {
	stateArchivalSettings, ok := c.configSettingEntry.GetStateArchivalSettings()
	if !ok {
		return 0
	}

	return uint32(stateArchivalSettings.MaxEntryTtl)
}

func (c ConfigSettingEntry) MinTemporaryTtl() uint32 {
	stateArchivalSettings, ok := c.configSettingEntry.GetStateArchivalSettings()
	if !ok {
		return 0
	}

	return uint32(stateArchivalSettings.MinTemporaryTtl)
}

func (c ConfigSettingEntry) MinPersistentTtl() uint32 {
	stateArchivalSettings, ok := c.configSettingEntry.GetStateArchivalSettings()
	if !ok {
		return 0
	}

	return uint32(stateArchivalSettings.MinPersistentTtl)
}

func (c ConfigSettingEntry) PersistentRentRateDenominator() int64 {
	stateArchivalSettings, ok := c.configSettingEntry.GetStateArchivalSettings()
	if !ok {
		return 0
	}

	return int64(stateArchivalSettings.PersistentRentRateDenominator)
}

func (c ConfigSettingEntry) TempRentRateDenominator() int64 {
	stateArchivalSettings, ok := c.configSettingEntry.GetStateArchivalSettings()
	if !ok {
		return 0
	}

	return int64(stateArchivalSettings.TempRentRateDenominator)
}

func (c ConfigSettingEntry) MaxEntriesToArchive() uint32 {
	stateArchivalSettings, ok := c.configSettingEntry.GetStateArchivalSettings()
	if !ok {
		return 0
	}

	return uint32(stateArchivalSettings.MaxEntriesToArchive)
}

func (c ConfigSettingEntry) BucketListSizeWindowSampleSize() uint32 {
	stateArchivalSettings, ok := c.configSettingEntry.GetStateArchivalSettings()
	if !ok {
		return 0
	}

	return uint32(stateArchivalSettings.BucketListSizeWindowSampleSize)
}

func (c ConfigSettingEntry) BucketListWindowSamplePeriod() uint32 {
	stateArchivalSettings, ok := c.configSettingEntry.GetStateArchivalSettings()
	if !ok {
		return 0
	}

	return uint32(stateArchivalSettings.BucketListWindowSamplePeriod)
}

func (c ConfigSettingEntry) EvictionScanSize() uint32 {
	stateArchivalSettings, ok := c.configSettingEntry.GetStateArchivalSettings()
	if !ok {
		return 0
	}

	return uint32(stateArchivalSettings.EvictionScanSize)
}

func (c ConfigSettingEntry) StartingEvictionScanLevel() uint32 {
	stateArchivalSettings, ok := c.configSettingEntry.GetStateArchivalSettings()
	if !ok {
		return 0
	}

	return uint32(stateArchivalSettings.StartingEvictionScanLevel)
}

func (c ConfigSettingEntry) LedgerMaxTxCount() uint32 {
	contractExecutionLanes, ok := c.configSettingEntry.GetContractExecutionLanes()
	if !ok {
		return 0
	}

	return uint32(contractExecutionLanes.LedgerMaxTxCount)
}

func (c ConfigSettingEntry) BucketListSizeWindow() []uint64 {
	bucketList, _ := c.configSettingEntry.GetBucketListSizeWindow()
	bucketListSizeWindow := make([]uint64, 0, len(bucketList))
	for _, sizeWindow := range bucketList {
		bucketListSizeWindow = append(bucketListSizeWindow, uint64(sizeWindow))
	}

	return bucketListSizeWindow
}

func (c ConfigSettingEntry) BucketListLevel() uint32 {
	evictionIterator, ok := c.configSettingEntry.GetEvictionIterator()
	if !ok {
		return 0
	}

	return uint32(evictionIterator.BucketListLevel)
}

func (c ConfigSettingEntry) IsCurrBucket() bool {
	evictionIterator, ok := c.configSettingEntry.GetEvictionIterator()
	if !ok {
		return false
	}

	return evictionIterator.IsCurrBucket
}

func (c ConfigSettingEntry) BucketFileOffset() uint64 {
	evictionIterator, ok := c.configSettingEntry.GetEvictionIterator()
	if !ok {
		return 0
	}

	return uint64(evictionIterator.BucketFileOffset)
}

func (c ConfigSettingEntry) Sponsor() string {
	return c.change.Sponsor()
}

func (c ConfigSettingEntry) LastModifiedLedger() uint32 {
	return c.change.LastModifiedLedger()
}

func (c ConfigSettingEntry) LedgerEntryChangeType() uint32 {
	return c.change.Type()
}

func (c ConfigSettingEntry) Deleted() bool {
	return c.change.Deleted()
}

func (c ConfigSettingEntry) ClosedAt() time.Time {
	return c.change.ClosedAt()
}

func (c ConfigSettingEntry) Sequence() uint32 {
	return c.change.Sequence()
}

func serializeCostParams(costParams xdr.ContractCostParams) []map[string]string {
	params := make([]map[string]string, 0, len(costParams))
	for _, contractCostParam := range costParams {
		serializedParam := map[string]string{}
		serializedParam["ConstTerm"] = strconv.Itoa(int(contractCostParam.ConstTerm))
		serializedParam["LinearTerm"] = strconv.Itoa(int(contractCostParam.LinearTerm))
		params = append(params, serializedParam)
	}

	return params
}
