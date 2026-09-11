package verification

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"math/bits"
	"sync"
)

// RFC 6962 Domain Separation prefixes.
const (
	RFC6962LeafHashPrefix = 0x00
	RFC6962NodeHashPrefix = 0x01
)

var (
	ErrIndexOutOfBounds = errors.New("transparency: index out of bounds")
	ErrInvalidTreeSize  = errors.New("transparency: invalid tree size")
	ErrInvalidProof     = errors.New("transparency: invalid proof")
)

// HashLeaf computes the RFC 6962 leaf hash: SHA-256(0x00 || data).
func HashLeaf(data []byte) [32]byte {
	h := sha256.New()
	h.Write([]byte{RFC6962LeafHashPrefix})
	h.Write(data)
	var out [32]byte
	copy(out[:], h.Sum(nil))
	return out
}

// HashChildren computes the RFC 6962 interior node hash: SHA-256(0x01 || left || right).
func HashChildren(left, right [32]byte) [32]byte {
	h := sha256.New()
	h.Write([]byte{RFC6962NodeHashPrefix})
	h.Write(left[:])
	h.Write(right[:])
	var out [32]byte
	copy(out[:], h.Sum(nil))
	return out
}

// TransparencyLog is an in-memory, thread-safe, RFC 6962 append-only Merkle tree.
type TransparencyLog struct {
	mu     sync.RWMutex
	leaves [][]byte
	hashes [][32]byte
}

// NewTransparencyLog creates an empty RFC 6962 append-only transparency log.
func NewTransparencyLog() *TransparencyLog {
	return &TransparencyLog{
		leaves: make([][]byte, 0),
		hashes: make([][32]byte, 0),
	}
}

// Size returns the current number of leaves in the log.
func (log *TransparencyLog) Size() int64 {
	log.mu.RLock()
	defer log.mu.RUnlock()
	return int64(len(log.leaves))
}

// AppendEntry appends an entry to the log, returning its zero-based leaf index
// and the updated Merkle tree root hash.
func (log *TransparencyLog) AppendEntry(data []byte) (int64, [32]byte, error) {
	log.mu.Lock()
	defer log.mu.Unlock()

	entryCopy := make([]byte, len(data))
	copy(entryCopy, data)
	leafHash := HashLeaf(entryCopy)

	log.leaves = append(log.leaves, entryCopy)
	log.hashes = append(log.hashes, leafHash)

	root := subTreeHash(log.hashes)
	return int64(len(log.leaves) - 1), root, nil
}

// RootHash returns the Merkle tree root hash for the full current log.
func (log *TransparencyLog) RootHash() [32]byte {
	log.mu.RLock()
	defer log.mu.RUnlock()
	return subTreeHash(log.hashes)
}

// RootHashAt returns the Merkle tree root hash for the tree of size treeSize.
func (log *TransparencyLog) RootHashAt(treeSize int64) ([32]byte, error) {
	log.mu.RLock()
	defer log.mu.RUnlock()

	if treeSize < 0 || treeSize > int64(len(log.hashes)) {
		return [32]byte{}, ErrInvalidTreeSize
	}
	return subTreeHash(log.hashes[:treeSize]), nil
}

// subTreeHash computes the RFC 6962 root hash of an arbitrary slice of leaf hashes.
func subTreeHash(hashes [][32]byte) [32]byte {
	n := int64(len(hashes))
	if n == 0 {
		return sha256.Sum256(nil)
	}
	if n == 1 {
		return hashes[0]
	}
	k := largestPowerOf2LessThan(n)
	left := subTreeHash(hashes[:k])
	right := subTreeHash(hashes[k:])
	return HashChildren(left, right)
}

func largestPowerOf2LessThan(n int64) int64 {
	if n <= 1 {
		return 0
	}
	leadingZeros := bits.LeadingZeros64(uint64(n - 1))
	return int64(1) << (63 - leadingZeros)
}

// InclusionProof returns the RFC 6962 audit path proving leafIndex is in a tree of size treeSize.
func (log *TransparencyLog) InclusionProof(leafIndex int64, treeSize int64) ([][32]byte, error) {
	log.mu.RLock()
	defer log.mu.RUnlock()

	if treeSize < 1 || treeSize > int64(len(log.hashes)) {
		return nil, ErrInvalidTreeSize
	}
	if leafIndex < 0 || leafIndex >= treeSize {
		return nil, ErrIndexOutOfBounds
	}

	return computeInclusionProof(log.hashes[:treeSize], leafIndex), nil
}

func computeInclusionProof(hashes [][32]byte, m int64) [][32]byte {
	n := int64(len(hashes))
	if n <= 1 {
		return nil
	}
	k := largestPowerOf2LessThan(n)
	if m < k {
		proof := computeInclusionProof(hashes[:k], m)
		rightHash := subTreeHash(hashes[k:])
		return append(proof, rightHash)
	}
	proof := computeInclusionProof(hashes[k:], m-k)
	leftHash := subTreeHash(hashes[:k])
	return append(proof, leftHash)
}

// VerifyInclusion checks that leaf with leafIndex is included in the tree of treeSize with root.
func VerifyInclusion(leaf []byte, leafIndex int64, treeSize int64, root [32]byte, proof [][32]byte) bool {
	if treeSize < 1 || leafIndex < 0 || leafIndex >= treeSize {
		return false
	}
	leafHash := HashLeaf(leaf)
	if treeSize == 1 {
		return len(proof) == 0 && leafHash == root
	}

	fn := leafIndex
	sn := treeSize - 1
	r := leafHash

	for _, p := range proof {
		if sn == 0 {
			return false
		}
		if fn%2 == 1 || fn == sn {
			r = HashChildren(p, r)
			for fn%2 == 0 && fn != 0 {
				fn /= 2
				sn /= 2
			}
		} else {
			r = HashChildren(r, p)
		}
		fn /= 2
		sn /= 2
	}
	return sn == 0 && r == root
}

// ConsistencyProof returns the RFC 6962 consistency path between treeSize1 and treeSize2.
func (log *TransparencyLog) ConsistencyProof(firstSize, secondSize int64) ([][32]byte, error) {
	log.mu.RLock()
	defer log.mu.RUnlock()

	total := int64(len(log.hashes))
	if firstSize < 1 || firstSize > secondSize || secondSize > total {
		return nil, ErrInvalidTreeSize
	}
	if firstSize == secondSize {
		return nil, nil
	}

	return computeSubProof(log.hashes[:secondSize], firstSize, true), nil
}

func computeSubProof(hashes [][32]byte, m int64, b bool) [][32]byte {
	n := int64(len(hashes))
	if m == n {
		if !b {
			return [][32]byte{subTreeHash(hashes)}
		}
		return nil
	}
	k := largestPowerOf2LessThan(n)
	if m <= k {
		proof := computeSubProof(hashes[:k], m, b)
		rightHash := subTreeHash(hashes[k:])
		return append(proof, rightHash)
	}
	proof := computeSubProof(hashes[k:], m-k, false)
	leftHash := subTreeHash(hashes[:k])
	return append(proof, leftHash)
}

// VerifyConsistency checks that the tree of size firstSize with root firstRoot
// is an older prefix of the tree of size secondSize with root secondRoot.
func VerifyConsistency(firstSize, secondSize int64, firstRoot, secondRoot [32]byte, proof [][32]byte) bool {
	if firstSize == secondSize {
		return len(proof) == 0 && bytes.Equal(firstRoot[:], secondRoot[:])
	}
	if firstSize < 1 || firstSize > secondSize {
		return false
	}
	if len(proof) == 0 {
		return false
	}

	proofIndex := 0
	var oldHash, newHash [32]byte

	if isPowerOf2(firstSize) {
		oldHash = firstRoot
		newHash = firstRoot
	} else {
		oldHash = proof[0]
		newHash = proof[0]
		proofIndex++
	}

	fn := firstSize - 1
	sn := secondSize - 1

	for fn%2 == 1 {
		fn /= 2
		sn /= 2
	}

	for ; sn > 0 && proofIndex < len(proof); proofIndex++ {
		p := proof[proofIndex]
		if fn%2 == 1 || fn == sn {
			oldHash = HashChildren(p, oldHash)
			newHash = HashChildren(p, newHash)
			for fn%2 == 0 && fn != 0 {
				fn /= 2
				sn /= 2
			}
		} else {
			newHash = HashChildren(newHash, p)
		}
		fn /= 2
		sn /= 2
	}

	return sn == 0 && proofIndex == len(proof) &&
		bytes.Equal(oldHash[:], firstRoot[:]) &&
		bytes.Equal(newHash[:], secondRoot[:])
}

func isPowerOf2(n int64) bool {
	return n > 0 && (n&(n-1)) == 0
}
