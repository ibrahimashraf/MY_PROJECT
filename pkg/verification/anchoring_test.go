package verification

import (
	"errors"
	"sync"
	"testing"
)

func TestAnchoringEngineAppendAuditInclusion(t *testing.T) {
	log := NewTransparencyLog()
	engine := NewAnchoringEngine(log)

	entries := []struct {
		typ     string
		payload []byte
	}{
		{EntryTypeStatutoryCertificate, []byte("CERT-A1")},
		{EntryTypeProofWitness, []byte("witness-88441")},
		{EntryTypeWorkOrder, []byte("WO-2026-0001")},
		{EntryTypePersonInChargeSigning, []byte("pic-sig-0001")},
		{EntryTypeStatutoryCertificate, []byte("CERT-B2")},
	}

	for i, e := range entries {
		idx, leafHash, err := engine.AppendEntry(e.typ, e.payload)
		if err != nil {
			t.Fatalf("append %d: %v", i, err)
		}
		if idx != int64(i) {
			t.Fatalf("append %d: expected index %d, got %d", i, i, idx)
		}
		if leafHash != HashLeaf(packLeaf(e.typ, e.payload)) {
			t.Fatalf("append %d: leaf hash mismatch", i)
		}
	}

	if engine.Size() != int64(len(entries)) {
		t.Fatalf("expected size %d, got %d", len(entries), engine.Size())
	}

	for i, e := range entries {
		proof, err := engine.GenerateAuditInclusionProof(int64(i))
		if err != nil {
			t.Fatalf("audit proof %d: %v", i, err)
		}
		if proof.LeafIndex != int64(i) {
			t.Fatalf("audit proof %d: wrong leaf index", i)
		}
		if !VerifyInclusionProof(proof, packLeaf(e.typ, e.payload)) {
			t.Fatalf("audit proof %d failed verification", i)
		}
		if VerifyInclusionProof(proof, append([]byte("tampered"), e.payload...)) {
			t.Fatalf("audit proof %d accepted tampered leaf", i)
		}
	}

	if _, err := engine.GenerateAuditInclusionProof(int64(len(entries))); !errors.Is(err, ErrLeafNotFound) {
		t.Fatalf("expected ErrLeafNotFound, got %v", err)
	}
}

func TestAnchoringEntryTypeDomainSeparation(t *testing.T) {
	engine := NewAnchoringEngine(NewTransparencyLog())
	payload := []byte("same-payload-bytes")

	i1, h1, err := engine.AppendEntry(EntryTypeStatutoryCertificate, payload)
	if err != nil {
		t.Fatal(err)
	}
	i2, h2, err := engine.AppendEntry(EntryTypeWorkOrder, payload)
	if err != nil {
		t.Fatal(err)
	}
	if h1 == h2 {
		t.Fatal("identical leaf hashes across entry types")
	}
	p1, err := engine.GenerateAuditInclusionProof(i1)
	if err != nil {
		t.Fatal(err)
	}
	p2, err := engine.GenerateAuditInclusionProof(i2)
	if err != nil {
		t.Fatal(err)
	}
	if !VerifyInclusionProof(p1, packLeaf(EntryTypeStatutoryCertificate, payload)) {
		t.Fatal("certificate leaf failed to verify")
	}
	if !VerifyInclusionProof(p2, packLeaf(EntryTypeWorkOrder, payload)) {
		t.Fatal("work order leaf failed to verify")
	}
}

func TestAnchoringConsistencyOverEvolvingSizes(t *testing.T) {
	engine := NewAnchoringEngine(NewTransparencyLog())
	const total = 12

	var states []ConsistencyProof
	for i := 1; i <= total; i++ {
		if _, _, err := engine.AppendEntry(EntryTypeWorkOrder, []byte("wo-entry")); err != nil {
			t.Fatal(err)
		}
	}
	// Re-append distinct payloads to exercise non-uniform leaves.
	for i := 0; i < total; i++ {
		if _, _, err := engine.AppendEntry(EntryTypeStatutoryCertificate, []byte{'a' + byte(i)}); err != nil {
			t.Fatal(err)
		}
	}

	for i := 1; i <= total; i++ {
		proof, err := engine.GenerateConsistencyProof(int64(i), int64(2*total))
		if err != nil {
			t.Fatalf("consistency %d: %v", i, err)
		}
		states = append(states, proof)
		if !VerifyConsistencyProof(proof) {
			t.Fatalf("consistency proof %d failed", i)
		}
	}
	for _, p := range states {
		if p.FirstSize == 0 || p.SecondSize != int64(2*total) || p.FirstRoot == [32]byte{} || p.SecondRoot == [32]byte{} {
			t.Fatalf("incomplete consistency proof: %+v", p)
		}
	}

	// Incremental first-box verification: each state's first root must hash
	// out to the second root as verified above; tamper one root and fail.
	bad := states[0]
	bad.SecondRoot[0] ^= 0xFF
	if VerifyConsistencyProof(bad) {
		t.Fatal("tampered consistency proof verified")
	}
}

func TestAnchoringRejectsInvalidState(t *testing.T) {
	engine := NewAnchoringEngine(NewTransparencyLog())
	if _, _, err := engine.AppendEntry("", []byte("x")); !errors.Is(err, ErrEmptyEntryType) {
		t.Fatalf("expected ErrEmptyEntryType, got %v", err)
	}
	if _, _, err := engine.AppendEntry("bad\x00type", []byte("x")); !errors.Is(err, ErrEntryTypeContainsNul) {
		t.Fatalf("expected ErrEntryTypeContainsNul, got %v", err)
	}
	if _, _, err := engine.AppendEntry(EntryTypeWorkOrder, []byte("x")); err != nil {
		t.Fatal(err)
	}
	if _, err := engine.GenerateConsistencyProof(0, 1); err == nil {
		t.Fatal("expected error for firstSize 0")
	}

	nilEngine := NewAnchoringEngine(nil)
	if _, _, err := nilEngine.AppendEntry(EntryTypeWorkOrder, []byte("x")); !errors.Is(err, ErrNilTransparencyLog) {
		t.Fatalf("expected ErrNilTransparencyLog, got %v", err)
	}
	if _, err := nilEngine.GenerateAuditInclusionProof(0); !errors.Is(err, ErrNilTransparencyLog) {
		t.Fatalf("expected ErrNilTransparencyLog for audit, got %v", err)
	}
}

func TestAnchoringConcurrentAppendAndAudit(t *testing.T) {
	engine := NewAnchoringEngine(NewTransparencyLog())
	const workers = 16
	const perWorker = 32

	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := 0; i < perWorker; i++ {
				payload := []byte(string(rune('A'+w)) + "entry")
				if _, _, err := engine.AppendEntry(EntryTypeProofWitness, payload); err != nil {
					t.Errorf("concurrent append: %v", err)
					return
				}
			}
		}(w)
	}
	wg.Wait()

	total := engine.Size()
	if total != workers*perWorker {
		t.Fatalf("expected %d entries, got %d", workers*perWorker, total)
	}
	for i := int64(0); i < total; i++ {
		proof, err := engine.GenerateAuditInclusionProof(i)
		if err != nil {
			t.Fatalf("audit proof %d: %v", i, err)
		}
		matches := 0
		for w := 0; w < workers; w++ {
			payload := []byte(string(rune('A'+w)) + "entry")
			if VerifyInclusionProof(proof, packLeaf(EntryTypeProofWitness, payload)) {
				matches++
			}
		}
		if matches != 1 {
			t.Fatalf("audit proof %d: expected exactly 1 matching payload, got %d", i, matches)
		}
	}
}
