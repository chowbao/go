package xdr

import "encoding/base64"

func (e *EncodingBuffer) claimableBalanceCompressEncodeTo(cb ClaimableBalanceId) error {
	if err := e.xdrEncoderBuf.WriteByte(byte(cb.Type)); err != nil {
		return err
	}
	switch cb.Type {
	case ClaimableBalanceIdTypeClaimableBalanceIdTypeV0:
		_, err := e.xdrEncoderBuf.Write(cb.V0[:])
		return err
	default:
		panic("Unknown type")
	}
}

// MarshalBinaryBase64 marshals XDR into a binary form and then encodes it
// using base64.
func (c ClaimableBalanceId) MarshalBinaryBase64() (string, error) {
	b, err := c.MarshalBinary()
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(b), nil
}
