package infotheory

import (
	"errors"
	"math"
)

// ShannonEntropy calculates the information entropy in bits: H(P) = -sum(p_i * log2(p_i)).
func ShannonEntropy(probs []float64) (float64, error) {
	if len(probs) == 0 {
		return 0, errors.New("probability distribution cannot be empty")
	}

	sum := 0.0
	for _, p := range probs {
		if p < 0 || p > 1.0 {
			return 0, errors.New("probabilities must be bounded in [0.0, 1.0]")
		}
		sum += p
	}

	if math.Abs(sum-1.0) > 1e-4 {
		return 0, errors.New("probabilities must sum to 1.0")
	}

	h := 0.0
	for _, p := range probs {
		if p > 1e-12 {
			h -= p * math.Log2(p)
		}
	}

	return h, nil
}

// KLDivergence computes relative entropy D_KL(P || Q) = sum(p_i * log2(p_i / q_i)).
func KLDivergence(p, q []float64) (float64, error) {
	if len(p) != len(q) || len(p) == 0 {
		return 0, errors.New("distributions p and q must have identical positive length")
	}

	kl := 0.0
	for i := 0; i < len(p); i++ {
		pi := p[i]
		qi := q[i]

		if pi > 1e-12 {
			if qi < 1e-12 {
				return math.Inf(1), nil // Infinite divergence if q has zero support
			}
			kl += pi * math.Log2(pi/qi)
		}
	}

	return kl, nil
}

// GF256 represents an element in the Galois Field GF(2^8).
type GF256 byte

const (
	// AES irreducible polynomial: x^8 + x^4 + x^3 + x + 1 (0x11B)
	IrreduciblePoly uint16 = 0x11B
)

// Add computes addition in GF(2^8) (equivalent to bitwise XOR).
func (a GF256) Add(b GF256) GF256 {
	return a ^ b
}

// Mul computes multiplication in GF(2^8) modulo the AES irreducible polynomial.
func (a GF256) Mul(b GF256) GF256 {
	var p uint8 = 0
	valA := uint8(a)
	valB := uint8(b)

	for i := 0; i < 8; i++ {
		if valB&1 != 0 {
			p ^= valA
		}
		highBit := valA & 0x80
		valA <<= 1
		if highBit != 0 {
			valA ^= 0x1B // Lower 8 bits of 0x11B
		}
		valB >>= 1
	}

	return GF256(p)
}

// Inverse computes the multiplicative inverse in GF(2^8) such that a * a^-1 = 1.
func (a GF256) Inverse() (GF256, error) {
	if a == 0 {
		return 0, errors.New("division by zero: element 0 has no multiplicative inverse in GF(2^8)")
	}

	// In GF(2^8), a^(255) = 1 (Fermat's Little Theorem) -> a^-1 = a^254
	var res GF256 = 1
	base := a
	exp := 254

	for exp > 0 {
		if exp&1 != 0 {
			res = res.Mul(base)
		}
		base = base.Mul(base)
		exp >>= 1
	}

	return res, nil
}
