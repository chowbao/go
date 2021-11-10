package benchmarks

import (
	"bytes"
	"encoding/base64"
	"testing"

	"github.com/stellar/go/gxdr"
	"github.com/stellar/go/xdr"
	goxdr "github.com/xdrpp/goxdr/xdr"
)

const input64 = "AAAAAgAAAACfHrX0tYB0gpXuJYTN9os06cdF62KAaqY9jid+777eyQAAC7gCM9czAAi/DQAAAAEAAAAAAAAAAAAAAABhga2dAAAAAAAAAAMAAAAAAAAADAAAAAAAAAABTU9CSQAAAAA8cTArnmXa4wEQJxDHOw5SwBaDVjBfAP5lRMNZkRtlZAAAAAAG42RBAAf7lQCYloAAAAAAMgbg0AAAAAAAAAADAAAAAU1PQkkAAAAAPHEwK55l2uMBECcQxzsOUsAWg1YwXwD+ZUTDWZEbZWQAAAAAAAAADkpyV7kAARBNABMS0AAAAAAyBuDRAAAAAAAAAAMAAAABTU9CSQAAAAA8cTArnmXa4wEQJxDHOw5SwBaDVjBfAP5lRMNZkRtlZAAAAAAAAAAclOSvewAIl5kAmJaAAAAAADIG4NIAAAAAAAAAAe++3skAAABAs2jt6+cyeyFvXVFphBcwt18GXnj7Jwa+hWQRyaBmPOSR2415GBi8XY3lC4m4aX9S322HvHjrxgQiar7KjgnQDw=="

var input = func() []byte {
	decoded, err := base64.StdEncoding.DecodeString(input64)
	if err != nil {
		panic(err)
	}
	return decoded
}()

var xdrInput = func() xdr.TransactionEnvelope {
	var te xdr.TransactionEnvelope
	if err := te.UnmarshalBinary(input); err != nil {
		panic(err)
	}
	return te
}()

var gxdrInput = func() gxdr.TransactionEnvelope {
	var te gxdr.TransactionEnvelope
	// note goxdr will panic if there's a marshaling error.
	te.XdrMarshal(&goxdr.XdrIn{In: bytes.NewReader(input)}, "")
	return te
}()

func BenchmarkXDRUnmarshalWithReflection(b *testing.B) {
	var (
		r  bytes.Reader
		te xdr.TransactionEnvelope
	)
	for i := 0; i < b.N; i++ {
		r.Reset(input)
		_, _ = xdr.Unmarshal(&r, &te)
	}
}

func BenchmarkXDRUnmarshal(b *testing.B) {
	var te xdr.TransactionEnvelope
	for i := 0; i < b.N; i++ {
		_ = te.UnmarshalBinary(input)
	}
}

func BenchmarkGXDRUnmarshal(b *testing.B) {
	var (
		te gxdr.TransactionEnvelope
		r  bytes.Reader
	)
	for i := 0; i < b.N; i++ {
		r.Reset(input)
		te.XdrMarshal(&goxdr.XdrIn{In: &r}, "")
	}
}

func BenchmarkXDRMarshalWithReflection(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = xdr.Marshal(&bytes.Buffer{}, xdrInput)
	}
}

func BenchmarkXDRMarshal(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = xdrInput.MarshalBinary()
	}
}

func BenchmarkXDRMarshalWithEncodingBuffer(b *testing.B) {
	e := xdr.NewEncodingBuffer()
	for i := 0; i < b.N; i++ {
		_, _ = e.UnsafeMarshalBinary(xdrInput)
	}
}

func BenchmarkGXDRMarshal(b *testing.B) {
	var output bytes.Buffer
	// Benchmark.
	for i := 0; i < b.N; i++ {
		output.Reset()
		gxdrInput.XdrMarshal(&goxdr.XdrOut{Out: &output}, "")
	}
}

func BenchmarkXDRMarshalHex(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = xdr.MarshalHex(xdrInput)
	}
}

func BenchmarkXDRMarshalHexWithEncodingBuffer(b *testing.B) {
	e := xdr.NewEncodingBuffer()
	for i := 0; i < b.N; i++ {
		_, _ = e.MarshalHex(xdrInput)
	}
}

func BenchmarkXDRUnsafeMarshalHexWithEncodingBuffer(b *testing.B) {
	e := xdr.NewEncodingBuffer()
	for i := 0; i < b.N; i++ {
		_, _ = e.UnsafeMarshalHex(xdrInput)
	}
}

func BenchmarkXDRMarshalBase64(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = xdr.MarshalBase64(xdrInput)
	}
}

func BenchmarkXDRMarshalBase64WithEncodingBuffer(b *testing.B) {
	e := xdr.NewEncodingBuffer()
	for i := 0; i < b.N; i++ {
		_, _ = e.MarshalBase64(xdrInput)
	}
}

func BenchmarkXDRUnsafeMarshalBase64WithEncodingBuffer(b *testing.B) {
	e := xdr.NewEncodingBuffer()
	for i := 0; i < b.N; i++ {
		_, _ = e.UnsafeMarshalBase64(xdrInput)
	}
}