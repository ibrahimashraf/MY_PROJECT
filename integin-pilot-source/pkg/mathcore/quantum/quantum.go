package quantum

import (
	"errors"
	"math"
	"math/cmplx"
)

// StateVector represents a multi-qubit quantum state vector of dimension 2^N.
type StateVector struct {
	NumQubits int
	Amps      []complex128
}

// NewQubitState creates a single-qubit state initialized to |0>.
func NewQubitState() *StateVector {
	return &StateVector{
		NumQubits: 1,
		Amps:      []complex128{complex(1, 0), complex(0, 0)},
	}
}

// NewMultiQubitState initializes an N-qubit state to |00...0>.
func NewMultiQubitState(n int) *StateVector {
	dim := 1 << n
	amps := make([]complex128, dim)
	amps[0] = complex(1, 0)
	return &StateVector{
		NumQubits: n,
		Amps:      amps,
	}
}

// NormSquared computes total state probability (must sum to 1.0).
func (sv *StateVector) NormSquared() float64 {
	sum := 0.0
	for _, a := range sv.Amps {
		sum += real(a)*real(a) + imag(a)*imag(a)
	}
	return sum
}

// Normalize normalizes the state vector to unity.
func (sv *StateVector) Normalize() {
	norm := math.Sqrt(sv.NormSquared())
	if norm > 1e-12 {
		invNorm := complex(1.0/norm, 0)
		for i := range sv.Amps {
			sv.Amps[i] *= invNorm
		}
	}
}

// ApplyHadamard applies a Hadamard gate to target qubit, creating superposition.
func (sv *StateVector) ApplyHadamard(targetQubit int) error {
	if targetQubit < 0 || targetQubit >= sv.NumQubits {
		return errors.New("target qubit index out of bounds")
	}

	invSqrt2 := complex(1.0/math.Sqrt(2), 0)
	step := 1 << targetQubit

	for i := 0; i < len(sv.Amps); i += 2 * step {
		for j := 0; j < step; j++ {
			idx0 := i + j
			idx1 := idx0 + step

			u := sv.Amps[idx0]
			v := sv.Amps[idx1]

			sv.Amps[idx0] = invSqrt2 * (u + v)
			sv.Amps[idx1] = invSqrt2 * (u - v)
		}
	}
	return nil
}

// ApplyPauliX applies a bit-flip NOT gate (|0> <-> |1>).
func (sv *StateVector) ApplyPauliX(targetQubit int) error {
	if targetQubit < 0 || targetQubit >= sv.NumQubits {
		return errors.New("target qubit index out of bounds")
	}
	step := 1 << targetQubit
	for i := 0; i < len(sv.Amps); i += 2 * step {
		for j := 0; j < step; j++ {
			idx0 := i + j
			idx1 := idx0 + step
			sv.Amps[idx0], sv.Amps[idx1] = sv.Amps[idx1], sv.Amps[idx0]
		}
	}
	return nil
}

// ApplyCNOT applies a Controlled-NOT gate between control and target qubits.
func (sv *StateVector) ApplyCNOT(control, target int) error {
	if control == target || control >= sv.NumQubits || target >= sv.NumQubits {
		return errors.New("invalid control or target qubit index")
	}

	for i := 0; i < len(sv.Amps); i++ {
		controlBit := (i >> control) & 1
		targetBit := (i >> target) & 1

		if controlBit == 1 && targetBit == 0 {
			flippedIdx := i ^ (1 << target)
			sv.Amps[i], sv.Amps[flippedIdx] = sv.Amps[flippedIdx], sv.Amps[i]
		}
	}
	return nil
}

// Probability returns the measurement probability of observing state basis idx.
func (sv *StateVector) Probability(basisIdx int) float64 {
	if basisIdx < 0 || basisIdx >= len(sv.Amps) {
		return 0
	}
	a := sv.Amps[basisIdx]
	return cmplx.Abs(a) * cmplx.Abs(a)
}
