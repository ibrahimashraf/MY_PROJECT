package rustfs

import (
	"bytes"
	"crypto/rand"
	"testing"
)

func TestCAS_ChunkDeduplicationAndReassembly(t *testing.T) {
	cas := NewMemoryCAS()
	chunkSize := 1024 // 1 KB for test

	// Create 2.5 KB of test data
	originalData := make([]byte, 2500)
	_, _ = rand.Read(originalData)

	// 1. Chunk and store stream
	manifest1, err := ChunkStream(bytes.NewReader(originalData), cas, chunkSize)
	if err != nil {
		t.Fatalf("failed chunking stream: %v", err)
	}

	if manifest1.TotalBytes != 2500 {
		t.Fatalf("expected 2500 bytes, got %d", manifest1.TotalBytes)
	}
	if len(manifest1.Chunks) != 3 { // 1024 + 1024 + 452
		t.Fatalf("expected 3 chunks, got %d", len(manifest1.Chunks))
	}

	// Initial store should have 3 unique blocks
	if len(cas.blocks) != 3 {
		t.Fatalf("expected 3 blocks in CAS, got %d", len(cas.blocks))
	}

	// 2. Reassemble stream and assert byte-for-byte fidelity
	var reassembled bytes.Buffer
	if err := ReassembleStream(manifest1, cas, &reassembled); err != nil {
		t.Fatalf("reassembly failed: %v", err)
	}

	if !bytes.Equal(originalData, reassembled.Bytes()) {
		t.Fatalf("reassembled bytes do not match original payload")
	}

	// 3. Delta mutation test: mutate only the middle chunk (offset 1024..2048)
	mutatedData := make([]byte, len(originalData))
	copy(mutatedData, originalData)
	mutatedData[1100] = ^mutatedData[1100] // Flip single byte in chunk 1

	manifest2, err := ChunkStream(bytes.NewReader(mutatedData), cas, chunkSize)
	if err != nil {
		t.Fatalf("failed chunking mutated stream: %v", err)
	}

	// Only 1 new chunk was created! Total blocks should now be 3 + 1 = 4
	if len(cas.blocks) != 4 {
		t.Fatalf("expected 4 blocks after single-chunk mutation, got %d (deduplication failed)", len(cas.blocks))
	}

	// Assert chunk 0 and chunk 2 have identical SHA-256 hashes
	if manifest1.Chunks[0].SHA256 != manifest2.Chunks[0].SHA256 {
		t.Fatalf("chunk 0 should be identical across revisions")
	}
	if manifest1.Chunks[2].SHA256 != manifest2.Chunks[2].SHA256 {
		t.Fatalf("chunk 2 should be identical across revisions")
	}
	if manifest1.Chunks[1].SHA256 == manifest2.Chunks[1].SHA256 {
		t.Fatalf("chunk 1 should have differed due to flipped byte")
	}
}
