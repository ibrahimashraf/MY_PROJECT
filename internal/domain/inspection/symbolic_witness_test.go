package inspection

import (
	"math"
	"strings"
	"testing"
)

func TestCertifyFindingCompliancePassesWithinTolerance(t *testing.T) {
	finding := Finding{ID: "finding-1", MeasuredValue: floatPtr(9.85), MeasuredUnit: "mm"}
	witness := CertifyFindingCompliance(finding, 0.010, 1e-3)
	if !witness.IsValid {
		t.Fatalf("expected compliance witness to be valid: %s", witness.Proof)
	}
	if math.Abs(witness.Residual-0.00015) > 1e-12 {
		t.Fatalf("residual = %g, want 0.00015", witness.Residual)
	}
}

func TestCertifyFindingComplianceBoundaryIsValid(t *testing.T) {
	finding := Finding{ID: "finding-1", MeasuredValue: floatPtr(0.75), MeasuredUnit: "m"}
	witness := CertifyFindingCompliance(finding, 0.5, 0.25)
	if !witness.IsValid {
		t.Fatalf("boundary residual must satisfy <= tolerance: %s", witness.Proof)
	}
}

func TestCertifyFindingComplianceFailsOutOfTolerance(t *testing.T) {
	finding := Finding{ID: "finding-1", MeasuredValue: floatPtr(10.5), MeasuredUnit: "mm"}
	witness := CertifyFindingCompliance(finding, 0.010, 1e-4)
	if witness.IsValid {
		t.Fatal("out-of-tolerance measurement must fail closed")
	}
	if math.Abs(witness.Residual-0.0005) > 1e-12 {
		t.Fatalf("residual = %g, want 0.0005", witness.Residual)
	}
}

func TestCertifyFindingComplianceFailsClosed(t *testing.T) {
	t.Run("missing measurement", func(t *testing.T) {
		witness := CertifyFindingCompliance(Finding{ID: "finding-1"}, 0.010, 1e-3)
		if witness.IsValid {
			t.Fatal("finding without measurement must fail closed")
		}
	})
	t.Run("invalid unit", func(t *testing.T) {
		finding := Finding{ID: "finding-1", MeasuredValue: floatPtr(10), MeasuredUnit: "watts"}
		witness := CertifyFindingCompliance(finding, 0.010, 1e-3)
		if witness.IsValid {
			t.Fatal("finding with unrecognized unit must fail closed")
		}
	})
	t.Run("negative tolerance", func(t *testing.T) {
		finding := Finding{ID: "finding-1", MeasuredValue: floatPtr(10), MeasuredUnit: "mm"}
		witness := CertifyFindingCompliance(finding, 0.010, -1e-3)
		if witness.IsValid {
			t.Fatal("negative tolerance must fail closed")
		}
	})
	t.Run("non-finite allowable", func(t *testing.T) {
		finding := Finding{ID: "finding-1", MeasuredValue: floatPtr(10), MeasuredUnit: "mm"}
		witness := CertifyFindingCompliance(finding, math.NaN(), 1e-3)
		if witness.IsValid {
			t.Fatal("NaN allowable must fail closed")
		}
	})
	t.Run("non-finite tolerance", func(t *testing.T) {
		finding := Finding{ID: "finding-1", MeasuredValue: floatPtr(10), MeasuredUnit: "mm"}
		witness := CertifyFindingCompliance(finding, 0.010, math.Inf(1))
		if witness.IsValid {
			t.Fatal("infinite tolerance must fail closed")
		}
	})
}

func TestCertifyFindingComplianceProducesReadableWitness(t *testing.T) {
	finding := Finding{ID: "finding-1", MeasuredValue: floatPtr(12), MeasuredUnit: "mm"}
	witness := CertifyFindingCompliance(finding, 0.010, 1e-3)
	if strings.TrimSpace(witness.Claim) == "" || strings.TrimSpace(witness.Proof) == "" {
		t.Fatal("witness claim and proof must be populated")
	}
	if !strings.Contains(witness.Proof, "m") || !strings.Contains(witness.Proof, "residual") {
		t.Fatalf("proof must state units and residual: %s", witness.Proof)
	}
}
