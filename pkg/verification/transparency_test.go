package verification

import (
	"bytes"
	"fmt"
	"testing"
)

func TestTransparencyLogAppendAndInclusionProof(t *testing.T) {
	log := NewTransparencyLog()

	entries := [][]byte{
		[]byte("certificate-001"),
		[]byte("certificate-002"),
		[]byte("certificate-003"),
		[]byte("certificate-004"),
		[]byte("certificate-005"),
		[]byte("certificate-006"),
		[]byte("certificate-007"),
	}

	for i, entry := range entries {
		idx, root, err := log.AppendEntry(entry)
		if err != nil {
			t.Fatalf("failed to append entry %d: %v", i, err)
		}
		if idx != int64(i) {
			t.Fatalf("expected leaf index %d, got %d", i, idx)
		}
		if root == [32]byte{} {
			t.Fatalf("expected non-empty root hash at index %d", i)
		}
	}

	treeSize := log.Size()
	if treeSize != int64(len(entries)) {
		t.Fatalf("expected size %d, got %d", len(entries), treeSize)
	}

	root := log.RootHash()

	// Verify inclusion proof for every single leaf
	for i, entry := range entries {
		proof, err := log.InclusionProof(int64(i), treeSize)
		if err != nil {
			t.Fatalf("failed to get inclusion proof for %d: %v", i, err)
		}

		valid := VerifyInclusion(entry, int64(i), treeSize, root, proof)
		if !valid {
			t.Fatalf("inclusion verification failed for entry %d", i)
		}

		// Tampered entry should fail
		tampered := []byte(fmt.Sprintf("%s-tampered", string(entry)))
		if VerifyInclusion(tampered, int64(i), treeSize, root, proof) {
			t.Fatalf("tampered entry unexpectedly verified for entry %d", i)
		}

		// Wrong index should fail
		if i > 0 && VerifyInclusion(entry, int64(i-1), treeSize, root, proof) {
			t.Fatalf("wrong index unexpectedly verified for entry %d", i)
		}
	}
}

func TestTransparencyLogConsistencyProofs(t *testing.T) {
	log := NewTransparencyLog()

	var roots [][32]byte

	for i := 1; i <= 10; i++ {
		entry := []byte(fmt.Sprintf("entry-%02d", i))
		_, root, err := log.AppendEntry(entry)
		if err != nil {
			t.Fatalf("failed to append entry %d: %v", i, err)
		}
		roots = append(roots, root)
	}

	// Verify consistency proof between every pair (i, j) where 1 <= i <= j <= 10
	for i := 1; i <= 10; i++ {
		for j := i; j <= 10; j++ {
			proof, err := log.ConsistencyProof(int64(i), int64(j))
			if err != nil {
				t.Fatalf("failed to get consistency proof from %d to %d: %v", i, j, err)
			}

			root1 := roots[i-1]
			root2 := roots[j-1]

			valid := VerifyConsistency(int64(i), int64(j), root1, root2, proof)
			if !valid {
				t.Fatalf("consistency verification failed between tree size %d and %d", i, j)
			}

			// Tampered root2 should fail
			if i < j {
				tamperedRoot := root2
				tamperedRoot[0] ^= 0xFF
				if VerifyConsistency(int64(i), int64(j), root1, tamperedRoot, proof) {
					t.Fatalf("tampered root unexpectedly verified between %d and %d", i, j)
				}
			}
		}
	}
}

func TestRFC6962DomainSeparation(t *testing.T) {
	data := []byte("test-data")
	leafHash := HashLeaf(data)

	// Interior node hash must use prefix 0x01
	nodeHash := HashChildren(leafHash, leafHash)

	if bytes.Equal(leafHash[:], nodeHash[:]) {
		t.Fatal("domain separation failure: leaf and node hash collided")
	}
}
