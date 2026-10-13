package inspection

import (
	"fmt"
	"math"

	"integin/pkg/engine/symbolic"
)

// CertifyFindingCompliance produces a formal proof witness for the claim
//
//	|measured - allowable| <= tolerance
//
// where measured is the finding's recording converted to canonical SI and
// allowable/tolerance are expressed in the same canonical SI units. Fails
// closed: missing, unrecognized, or non-finite measurements and non-finite or
// negative tolerances always yield IsValid=false.
func CertifyFindingCompliance(f Finding, allowable float64, tol float64) symbolic.ProofWitness {
	reject := func(reason string) symbolic.ProofWitness {
		return symbolic.ProofWitness{
			Claim:   fmt.Sprintf("finding %q compliance: |measured - allowable| <= %g", f.ID, tol),
			Proof:   "rejected: " + reason,
			IsValid: false,
		}
	}
	if math.IsNaN(allowable) || math.IsInf(allowable, 0) ||
		math.IsNaN(tol) || math.IsInf(tol, 0) || tol < 0 {
		return reject("allowable not finite or tolerance negative")
	}
	m, ok := f.MeasuredSI()
	if !ok {
		return reject("missing or non-finite/invalid unit measurement")
	}
	residual := math.Abs(m.Value - allowable)
	return symbolic.ProofWitness{
		Claim:    fmt.Sprintf("finding %q compliance: |measured - allowable| <= %g (%s)", f.ID, tol, m.Unit),
		Proof:    fmt.Sprintf("measured=%g %s, allowable=%g %s, residual=%g", m.Value, m.Unit, allowable, m.Unit, residual),
		IsValid:  residual <= tol,
		Residual: residual,
	}
}
