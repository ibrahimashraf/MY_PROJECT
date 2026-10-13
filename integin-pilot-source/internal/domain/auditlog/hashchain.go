package auditlog

import (
	"crypto/sha256"
	"fmt"
)

const genesisHash = "0000000000000000000000000000000000000000000000000000000000000000"

func GenesisHash() string {
	return genesisHash
}

func ComputeHash(previousHash, entryData string) string {
	h := sha256.Sum256([]byte(previousHash + entryData))
	return fmt.Sprintf("%x", h)
}
