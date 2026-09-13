package verification

import (
	"errors"
	"strings"
	"sync"
)

// Entry types connect INTEGIN domain artefacts to the RFC 6962
// Certificate Transparency log (Pillar IV ledger).
const (
	EntryTypePersonInChargeSigning = "person-in-charge-signing"
	EntryTypeStatutoryCertificate  = "statutory-certificate"
	EntryTypeProofWitness          = "proof-witness"
	EntryTypeWorkOrder             = "work-order"
)

var (
	ErrNilTransparencyLog   = errors.New("anchoring: nil transparency log")
	ErrEmptyEntryType       = errors.New("anchoring: empty entry type")
	ErrEntryTypeContainsNul = errors.New("anchoring: entry type contains NUL")
	ErrLeafNotFound         = errors.New("anchoring: no leaf recorded at index")
)

// anchorLeafHash carries the leaf hash recorded at one log index so audit
// proofs can re-assert the exact appended leaf without re-reading payload.
type anchorLeafHash struct {
	hash [32]byte
}

// AnchoringEngine is the sole write-through gateway to a TransparencyLog,
// domain-separating entries by type so identical payloads under different
// types produce distinct leaves.
type AnchoringEngine struct {
	mu         sync.RWMutex
	log        *TransparencyLog
	leafHashes []anchorLeafHash
}

// NewAnchoringEngine wraps an RFC 6962 TransparencyLog. Passing nil yields an
// engine that returns ErrNilTransparencyLog on every operation.
func NewAnchoringEngine(log *TransparencyLog) *AnchoringEngine {
	return &AnchoringEngine{log: log}
}

// packLeaf deterministically namespaces the payload under its entry type:
// entryType || 0x00 || payload. The NUL separator keeps the type boundary
// unambiguous regardless of payload bytes.
func packLeaf(entryType string, payload []byte) []byte {
	leaf := make([]byte, 0, len(entryType)+1+len(payload))
	leaf = append(leaf, entryType...)
	leaf = append(leaf, 0x00)
	leaf = append(leaf, payload...)
	return leaf
}

// AppendEntry appends a typed entry to the transparency log and returns its
// zero-based leaf index and the RFC 6962 leaf hash (SHA-256(0x00 || leaf)).
func (e *AnchoringEngine) AppendEntry(entryType string, payload []byte) (int64, [32]byte, error) {
	if e.log == nil {
		return 0, [32]byte{}, ErrNilTransparencyLog
	}
	if entryType == "" {
		return 0, [32]byte{}, ErrEmptyEntryType
	}
	if strings.ContainsRune(entryType, 0x00) {
		return 0, [32]byte{}, ErrEntryTypeContainsNul
	}
	leaf := packLeaf(entryType, payload)

	e.mu.Lock()
	defer e.mu.Unlock()
	idx, _, err := e.log.AppendEntry(leaf)
	if err != nil {
		return 0, [32]byte{}, err
	}
	leafHash := HashLeaf(leaf)
	e.leafHashes = append(e.leafHashes, anchorLeafHash{hash: leafHash})
	return idx, leafHash, nil
}

// InclusionProof is an RFC 6962 audit path for a single leaf.
type InclusionProof struct {
	LeafIndex int64
	TreeSize  int64
	LeafHash  [32]byte
	RootHash  [32]byte
	Path      [][32]byte
}

// GenerateAuditInclusionProof returns the audited inclusion proof for a
// previously appended leaf against the current tree root.
func (e *AnchoringEngine) GenerateAuditInclusionProof(leafIndex int64) (InclusionProof, error) {
	if e.log == nil {
		return InclusionProof{}, ErrNilTransparencyLog
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	if leafIndex < 0 || leafIndex >= int64(len(e.leafHashes)) {
		return InclusionProof{}, ErrLeafNotFound
	}
	treeSize := e.log.Size()
	path, err := e.log.InclusionProof(leafIndex, treeSize)
	if err != nil {
		return InclusionProof{}, err
	}
	return InclusionProof{
		LeafIndex: leafIndex,
		TreeSize:  treeSize,
		LeafHash:  e.leafHashes[leafIndex].hash,
		RootHash:  e.log.RootHash(),
		Path:      path,
	}, nil
}

// VerifyInclusionProof checks an audited inclusion proof against the included
// leaf data bytes. The leaf bytes must be the exact payload appended.
func VerifyInclusionProof(proof InclusionProof, leafData []byte) bool {
	return VerifyInclusion(leafData, proof.LeafIndex, proof.TreeSize, proof.RootHash, proof.Path)
}

// ConsistencyProof binds the roots of two tree sizes to an RFC 6962
// consistency path over evolving log sizes.
type ConsistencyProof struct {
	FirstSize  int64
	SecondSize int64
	FirstRoot  [32]byte
	SecondRoot [32]byte
	Path       [][32]byte
}

// GenerateConsistencyProof returns the consistency proof and both roots for
// firstSize < secondSize.
func (e *AnchoringEngine) GenerateConsistencyProof(firstSize, secondSize int64) (ConsistencyProof, error) {
	if e.log == nil {
		return ConsistencyProof{}, ErrNilTransparencyLog
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	firstRoot, err := e.log.RootHashAt(firstSize)
	if err != nil {
		return ConsistencyProof{}, err
	}
	secondRoot, err := e.log.RootHashAt(secondSize)
	if err != nil {
		return ConsistencyProof{}, err
	}
	path, err := e.log.ConsistencyProof(firstSize, secondSize)
	if err != nil {
		return ConsistencyProof{}, err
	}
	return ConsistencyProof{
		FirstSize:  firstSize,
		SecondSize: secondSize,
		FirstRoot:  firstRoot,
		SecondRoot: secondRoot,
		Path:       path,
	}, nil
}

// VerifyConsistencyProof checks the consistency evidence between two tree
// sizes.
func VerifyConsistencyProof(proof ConsistencyProof) bool {
	if proof.FirstSize == proof.SecondSize {
		return len(proof.Path) == 0 && proof.FirstRoot == proof.SecondRoot
	}
	return VerifyConsistency(proof.FirstSize, proof.SecondSize, proof.FirstRoot, proof.SecondRoot, proof.Path)
}

// RootHash returns the Merkle root of the full current log.
func (e *AnchoringEngine) RootHash() [32]byte {
	if e.log == nil {
		return [32]byte{}
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.log.RootHash()
}

// RootHashAt returns the Merkle root for a tree of treeSize leaves.
func (e *AnchoringEngine) RootHashAt(treeSize int64) ([32]byte, error) {
	if e.log == nil {
		return [32]byte{}, ErrNilTransparencyLog
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.log.RootHashAt(treeSize)
}

// Size returns the current number of appended leaves.
func (e *AnchoringEngine) Size() int64 {
	if e.log == nil {
		return 0
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.log.Size()
}
