package domain

import (
	"errors"
	"fmt"
)

var (
	ErrNotAssetDID   = errors.New("did is not an asset did")
	ErrAssetNotFound = errors.New("asset passport not found")
)

// ResolveAssetDID parses a raw DID, enforces that it is an asset DID, and
// resolves the identifier through the provided lookup callback. No network.
func ResolveAssetDID(raw string, lookup func(identifier string) (*UniversalAssetPassport, error)) (*UniversalAssetPassport, error) {
	did, err := ParseDID(raw)
	if err != nil {
		return nil, err
	}
	if did.Type != DIDTypeAsset {
		return nil, fmt.Errorf("%w: %s", ErrNotAssetDID, did.Type)
	}
	passport, err := lookup(did.Identifier)
	if err != nil {
		return nil, err
	}
	if passport == nil {
		return nil, ErrAssetNotFound
	}
	return passport, nil
}
