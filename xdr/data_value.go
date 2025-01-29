package xdr

import "encoding/base64"

// MarshalBinaryBase64 marshals XDR into a binary form and then encodes it
// using base64.
func (d DataValue) MarshalBinaryBase64() (string, error) {
	b, err := d.MarshalBinary()
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(b), nil
}
