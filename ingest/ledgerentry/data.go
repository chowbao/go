package ledgerentry

import "github.com/stellar/go/xdr"

type Data struct {
	AccountID string `json:"account_id"`
	DataName  string `json:"data_name"`
	DataValue string `json:"data_value"`
}

func DataDetails(dataEntry *xdr.DataEntry) (Data, error) {
	data := Data{
		AccountID: dataEntry.AccountId.Address(),
		DataName:  string(dataEntry.DataName),
	}

	dataValue, err := xdr.MarshalBase64(dataEntry.DataValue)
	if err != nil {
		return Data{}, err
	}

	data.DataValue = dataValue

	return data, nil
}
