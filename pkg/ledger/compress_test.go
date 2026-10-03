package ledger_test

import (
	"bytes"
	"testing"

	"integin/pkg/ledger"
)

func TestLedger_CompressZstd(t *testing.T) {
	raw := []byte("canonical-64-byte-merkle-crdt-audit-record-stream-test-payload")
	comp, err := ledger.CompressZstd(raw)
	if err != nil {
		t.Fatalf("CompressZstd failed: %v", err)
	}

	decomp, err := ledger.DecompressZstd(comp)
	if err != nil {
		t.Fatalf("DecompressZstd failed: %v", err)
	}

	if !bytes.Equal(decomp, raw) {
		t.Fatalf("decompressed mismatch: expected %q, got %q", raw, decomp)
	}
}
