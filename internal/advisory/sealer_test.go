package advisory

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"
	"time"
)

func fixedTime() time.Time {
	return time.Date(2026, 9, 11, 0, 0, 0, 0, time.UTC)
}

func TestSealGoldenVector(t *testing.T) {
	weights := strings.NewReader("weights")
	prompt := "What is the load rating?"
	tensors := []byte{0x01, 0x02, 0x03}
	rec, err := Seal(weights, prompt, tensors, "model-v1", fixedTime(), "")
	if err != nil {
		t.Fatal(err)
	}
	if rec.ModelID != "model-v1" {
		t.Fatalf("model id: got %q", rec.ModelID)
	}
	if rec.PrevSeal != sealGenesisHash {
		t.Fatalf("prev seal should be genesis, got %q", rec.PrevSeal)
	}
	if rec.SealHash == "" {
		t.Fatal("seal hash must not be empty")
	}
	if len(rec.SealHash) != 64 {
		t.Fatalf("seal hash should be 64 hex chars, got %d", len(rec.SealHash))
	}
	if rec.WeightsHash != fmt.Sprintf("%x", sha256.Sum256([]byte("weights"))) {
		t.Fatalf("weights hash mismatch")
	}
	if rec.PromptHash != fmt.Sprintf("%x", sha256.Sum256([]byte(prompt))) {
		t.Fatalf("prompt hash mismatch")
	}
	if rec.TensorHash != fmt.Sprintf("%x", sha256.Sum256(tensors)) {
		t.Fatalf("tensor hash mismatch")
	}
	const goldenSealHash = "00a70936537310723b6f557ee715885d7fe6131d209fcb68c61d79f65ae9f488"
	if rec.SealHash != goldenSealHash {
		t.Fatalf("golden seal hash mismatch: got %q want %q", rec.SealHash, goldenSealHash)
	}
}

func TestSealGoldenVectorDeterministic(t *testing.T) {
	weights := strings.NewReader("weights")
	prompt := "What is the load rating?"
	tensors := []byte{0x01, 0x02, 0x03}
	first, err := Seal(weights, prompt, tensors, "model-v1", fixedTime(), "")
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 100; i++ {
		r, err := Seal(strings.NewReader("weights"), prompt, tensors, "model-v1", fixedTime(), "")
		if err != nil {
			t.Fatal(err)
		}
		if r.SealHash != first.SealHash {
			t.Fatalf("iteration %d: seal hash differs: %q vs %q", i, r.SealHash, first.SealHash)
		}
	}
}

func TestSealChainVerify(t *testing.T) {
	evals := []string{"eval-A", "eval-B", "eval-C"}
	var chain []SealRecord
	var prev string
	for _, e := range evals {
		rec, err := Seal(strings.NewReader("w"), e, []byte{1}, "m", fixedTime(), prev)
		if err != nil {
			t.Fatal(err)
		}
		chain = append(chain, rec)
		prev = rec.SealHash
	}
	if err := VerifyChain(chain); err != nil {
		t.Fatalf("valid chain should verify: %v", err)
	}
}

func TestSealChainTamperFails(t *testing.T) {
	rec1, _ := Seal(strings.NewReader("w"), "prompt1", []byte{1}, "m", fixedTime(), "")
	rec2, _ := Seal(strings.NewReader("w"), "prompt2", []byte{2}, "m", fixedTime(), rec1.SealHash)
	chain := []SealRecord{rec1, rec2}
	if err := VerifyChain(chain); err != nil {
		t.Fatal(err)
	}
	rec2.TensorHash = rec2.TensorHash[:1] + "f" + rec2.TensorHash[2:]
	chain[1] = rec2
	if err := VerifyChain(chain); err == nil {
		t.Fatal("tampered chain should fail")
	}
}

func TestSealTamperTensorByteFailsVerifyArtifacts(t *testing.T) {
	tensors := []byte{0xAA, 0xBB, 0xCC}
	rec, _ := Seal(strings.NewReader("w"), "prompt1", tensors, "m", fixedTime(), "")
	if err := VerifyArtifacts(rec, strings.NewReader("w"), "prompt1", tensors); err != nil {
		t.Fatalf("valid artifacts should verify: %v", err)
	}
	tampered := []byte{0xAA, 0xBB, 0xDD}
	if err := VerifyArtifacts(rec, strings.NewReader("w"), "prompt1", tampered); err == nil {
		t.Fatal("one flipped tensor byte should fail artifact verify")
	}
	if err := VerifyArtifacts(rec, strings.NewReader("x"), "prompt1", tensors); err == nil {
		t.Fatal("mutated weights should fail artifact verify")
	}
	if err := VerifyArtifacts(rec, strings.NewReader("w"), "prompt2", tensors); err == nil {
		t.Fatal("mutated prompt should fail artifact verify")
	}
}

func TestSealEmptyWeightsRejects(t *testing.T) {
	_, err := Seal(strings.NewReader(""), "prompt", []byte{1}, "m", fixedTime(), "")
	if err == nil {
		t.Fatal("empty weights should be rejected")
	}
}

func TestSealEmptyPromptRejects(t *testing.T) {
	_, err := Seal(strings.NewReader("w"), "", []byte{1}, "m", fixedTime(), "")
	if err == nil {
		t.Fatal("empty prompt should be rejected")
	}
}

func TestSealNilWeightsRejects(t *testing.T) {
	_, err := Seal(nil, "prompt", []byte{1}, "m", fixedTime(), "")
	if err == nil {
		t.Fatal("nil weights should be rejected")
	}
}

func TestSealEmptyModelIDRejects(t *testing.T) {
	_, err := Seal(strings.NewReader("w"), "prompt", []byte{1}, "", fixedTime(), "")
	if err == nil {
		t.Fatal("empty model id should be rejected")
	}
}

func TestSealZeroTimeRejects(t *testing.T) {
	_, err := Seal(strings.NewReader("w"), "prompt", []byte{1}, "m", time.Time{}, "")
	if err == nil {
		t.Fatal("zero time should be rejected")
	}
}

func TestSealEmptyTensorsRejects(t *testing.T) {
	_, err := Seal(strings.NewReader("w"), "prompt", []byte{}, "m", fixedTime(), "")
	if err == nil {
		t.Fatal("empty tensors should be rejected")
	}
}

func TestSealChainFirstRecordMustCommitGenesis(t *testing.T) {
	rec1, _ := Seal(strings.NewReader("w"), "prompt", []byte{1}, "m", fixedTime(), "not-genesis")
	chain := []SealRecord{rec1}
	if err := VerifyChain(chain); err == nil {
		t.Fatal("first record with non-genesis prev should fail")
	}
}

func TestSealChainBrokenLinkFails(t *testing.T) {
	rec1, _ := Seal(strings.NewReader("w"), "prompt1", []byte{1}, "m", fixedTime(), "")
	rec2, _ := Seal(strings.NewReader("w"), "prompt2", []byte{2}, "m", fixedTime(), "wrong-prev")
	chain := []SealRecord{rec1, rec2}
	if err := VerifyChain(chain); err == nil {
		t.Fatal("broken chain link should fail")
	}
}

func TestSealChainEmptyRejects(t *testing.T) {
	if err := VerifyChain(nil); err == nil {
		t.Fatal("empty chain should be rejected")
	}
	if err := VerifyChain([]SealRecord{}); err == nil {
		t.Fatal("empty chain should be rejected")
	}
}

func TestSealDeterminismMapOrder(t *testing.T) {
	for i := 0; i < 100; i++ {
		rec1, err := Seal(bytes.NewReader([]byte{0xDE, 0xAD, 0xBE, 0xEF}), "analyze these sensors: "+string(rune('A'+i%26)), []byte{0x01, 0x02, 0x03}, "model-det", fixedTime(), "")
		if err != nil {
			t.Fatal(err)
		}
		rec2, err := Seal(bytes.NewReader([]byte{0xDE, 0xAD, 0xBE, 0xEF}), "analyze these sensors: "+string(rune('A'+i%26)), []byte{0x01, 0x02, 0x03}, "model-det", fixedTime(), "")
		if err != nil {
			t.Fatal(err)
		}
		if rec1.SealHash != rec2.SealHash {
			t.Fatalf("iteration %d: determinism broken: %q vs %q", i, rec1.SealHash, rec2.SealHash)
		}
	}
}

func TestVerifySealSingleRecord(t *testing.T) {
	rec, _ := Seal(strings.NewReader("w"), "prompt", []byte{1}, "m", fixedTime(), "")
	if err := VerifySeal(rec); err != nil {
		t.Fatalf("valid single seal should verify: %v", err)
	}
}

func TestVerifySealTamperHashFails(t *testing.T) {
	rec, _ := Seal(strings.NewReader("w"), "prompt", []byte{1}, "m", fixedTime(), "")
	rec.SealHash = "0000000000000000000000000000000000000000000000000000000000000000"
	if err := VerifySeal(rec); err == nil {
		t.Fatal("tampered seal hash should fail")
	}
}
