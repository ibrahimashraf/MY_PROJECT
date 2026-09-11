package symbolic

import (
	"testing"
)

func TestTokenizer(t *testing.T) {
	tokens := Tokenize("stress_Pa = 355e6 * safety_factor")
	if len(tokens) < 5 {
		t.Fatalf("Expected at least 5 tokens, got %d", len(tokens))
	}
	if tokens[0].Type != TokenIdent || tokens[0].Literal != "stress_Pa" {
		t.Fatalf("First token should be identifier 'stress_Pa', got %v", tokens[0])
	}
	last := tokens[len(tokens)-1]
	if last.Type != TokenEOF {
		t.Fatal("Last token must be EOF")
	}
}

func TestProofWitness(t *testing.T) {
	// Claim: beam utilization <= 1.0, witness: computed value = 0.92
	witness := VerifyClaim("beam_utilization <= 1.0", "computed_utilization = 0.92", 0.92, 1.0)
	if !witness.IsValid {
		t.Fatal("Valid structural claim should be accepted by proof witness")
	}

	// Invalid: utilization = 1.15 > 1.0
	invalid := VerifyClaim("beam_utilization <= 1.0", "computed_utilization = 1.15", 1.15, 1.0)
	if invalid.IsValid {
		t.Fatal("Invalid structural claim should be rejected")
	}
}

func TestCurryHowardIdentityProof(t *testing.T) {
	proof := IdentityProof("StructuralSafety")
	if !proof.Valid {
		t.Fatal("Identity proof must always be valid")
	}
	if proof.TermType != "StructuralSafety → StructuralSafety" {
		t.Fatalf("Identity proof term type mismatch: %s", proof.TermType)
	}
}
