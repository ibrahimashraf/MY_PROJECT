package auditlog

import (
	"testing"
)

func TestGenesisHash(t *testing.T) {
	h := GenesisHash()
	if len(h) != 64 {
		t.Fatalf("expected 64 chars, got %d", len(h))
	}
	if h != "0000000000000000000000000000000000000000000000000000000000000000" {
		t.Fatalf("unexpected genesis hash: %s", h)
	}
}

func TestComputeHash(t *testing.T) {
	h := ComputeHash("abc", "def")
	if len(h) != 64 {
		t.Fatalf("expected 64 chars, got %d", len(h))
	}
}

func TestComputeHashDeterministic(t *testing.T) {
	h1 := ComputeHash("prev", "data")
	h2 := ComputeHash("prev", "data")
	if h1 != h2 {
		t.Fatalf("hash should be deterministic: %s != %s", h1, h2)
	}
}

func TestComputeHashDifferentInputs(t *testing.T) {
	h1 := ComputeHash("prev1", "data")
	h2 := ComputeHash("prev2", "data")
	if h1 == h2 {
		t.Fatal("different inputs should produce different hashes")
	}
}
