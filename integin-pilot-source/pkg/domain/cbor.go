package domain

import (
	"github.com/fxamacker/cbor/v2"
)

// ToCBOR serializes a UniversalAssetPassport into compact CBOR bytes (RFC 8949)
// for high-density QR passport rendering (did:integin).
func (p UniversalAssetPassport) ToCBOR() ([]byte, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}
	type rawPassport UniversalAssetPassport
	return cbor.Marshal(rawPassport(p))
}

// FromCBOR decodes a CBOR byte slice into a UniversalAssetPassport.
func FromCBOR(data []byte) (UniversalAssetPassport, error) {
	type rawPassport UniversalAssetPassport
	var raw rawPassport
	if err := cbor.Unmarshal(data, &raw); err != nil {
		return UniversalAssetPassport{}, err
	}
	p := UniversalAssetPassport(raw)
	if err := p.Validate(); err != nil {
		return UniversalAssetPassport{}, err
	}
	return p, nil
}
