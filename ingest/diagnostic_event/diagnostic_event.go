package diagnosticevent

import (
	"encoding/json"
	"fmt"

	"github.com/chowbao/go-stellar-xdr-json/xdr2json"
	"github.com/stellar/go/strkey"
	"github.com/stellar/go/xdr"
)

func Successful(d xdr.DiagnosticEvent) bool {
	return d.InSuccessfulContractCall
}

func Type(d xdr.DiagnosticEvent) int32 {
	return int32(d.Event.Type)
}

func ContractID(d xdr.DiagnosticEvent) (string, bool, error) {
	if d.Event.ContractId == nil {
		return "", false, nil
	}

	var err error
	var contractIdByte []byte
	var contractIDString string
	contractId := *d.Event.ContractId
	contractIdByte, err = contractId.MarshalBinary()
	if err != nil {
		return "", false, nil
	}
	contractIDString, err = strkey.Encode(strkey.VersionByteContract, contractIdByte)
	if err != nil {
		return "", false, nil
	}

	return contractIDString, true, nil
}

func Topics(d xdr.DiagnosticEvent) ([]interface{}, error) {
	topics, err := GetEventTopics(d)
	if err != nil {
		return []interface{}{}, err
	}

	return serializeScValArray(topics)
}

func Data(d xdr.DiagnosticEvent) (interface{}, error) {
	data, err := GetEventData(d)
	if err != nil {
		return []interface{}{}, err
	}

	return serializeScVal(data)
}

func GetEventTopics(d xdr.DiagnosticEvent) ([]xdr.ScVal, error) {
	switch d.Event.Body.V {
	case 0:
		contractEventV0 := d.Event.Body.MustV0()
		return contractEventV0.Topics, nil
	default:
		return []xdr.ScVal{}, fmt.Errorf("unsupported event body version: " + string(d.Event.Body.V))
	}
}

func GetEventData(d xdr.DiagnosticEvent) (xdr.ScVal, error) {
	switch d.Event.Body.V {
	case 0:
		contractEventV0 := d.Event.Body.MustV0()
		return contractEventV0.Data, nil
	default:
		return xdr.ScVal{}, fmt.Errorf("unsupported event body version: " + string(d.Event.Body.V))
	}
}

func serializeScVal(scVal xdr.ScVal) (interface{}, error) {
	var serializedDataDecoded interface{}
	serializedDataDecoded = "n/a"

	if _, ok := scVal.ArmForSwitch(int32(scVal.Type)); ok {
		var err error
		var raw []byte
		var jsonMessage json.RawMessage
		raw, err = scVal.MarshalBinary()
		if err != nil {
			return nil, err
		}

		jsonMessage, err = xdr2json.ConvertBytes(xdr.ScVal{}, raw)
		if err != nil {
			return nil, err
		}

		serializedDataDecoded = jsonMessage
	}

	return serializedDataDecoded, nil
}

func serializeScValArray(scVals []xdr.ScVal) ([]interface{}, error) {
	dataDecoded := make([]interface{}, 0, len(scVals))

	for _, scVal := range scVals {
		serializedDataDecoded, err := serializeScVal(scVal)
		if err != nil {
			return nil, err
		}
		dataDecoded = append(dataDecoded, serializedDataDecoded)
	}

	return dataDecoded, nil
}
