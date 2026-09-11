package discrete

import (
	"errors"
	"math/big"
)

// Factorial computes n! using arbitrary-precision big integers.
func Factorial(n int64) *big.Int {
	if n < 0 {
		return big.NewInt(0)
	}
	res := big.NewInt(1)
	for i := int64(2); i <= n; i++ {
		res.Mul(res, big.NewInt(i))
	}
	return res
}

// Combinations computes nCr = n! / (r! * (n-r)!).
func Combinations(n, r int64) (*big.Int, error) {
	if r < 0 || r > n {
		return nil, errors.New("invalid combination parameters: r must be in [0, n]")
	}
	if r == 0 || r == n {
		return big.NewInt(1), nil
	}
	// Optimize nCr symmetry: nCr = nC(n-r)
	if r > n-r {
		r = n - r
	}

	num := big.NewInt(1)
	den := big.NewInt(1)

	for i := int64(1); i <= r; i++ {
		num.Mul(num, big.NewInt(n-i+1))
		den.Mul(den, big.NewInt(i))
	}

	return new(big.Int).Quo(num, den), nil
}

// Permutations computes nPr = n! / (n-r)!.
func Permutations(n, r int64) (*big.Int, error) {
	if r < 0 || r > n {
		return nil, errors.New("invalid permutation parameters: r must be in [0, n]")
	}
	res := big.NewInt(1)
	for i := int64(0); i < r; i++ {
		res.Mul(res, big.NewInt(n-i))
	}
	return res, nil
}

// LogicOp defines boolean logic operators.
type LogicOp string

const (
	OpAnd     LogicOp = "AND"
	OpOr      LogicOp = "OR"
	OpNot     LogicOp = "NOT"
	OpImplies LogicOp = "IMPLIES"
	OpEquiv   LogicOp = "EQUIV"
)

// EvaluateLogic computes truth outcomes for elementary propositional logic.
func EvaluateLogic(op LogicOp, p, q bool) bool {
	switch op {
	case OpAnd:
		return p && q
	case OpOr:
		return p || q
	case OpNot:
		return !p
	case OpImplies:
		return !p || q // p -> q is equivalent to !p || q
	case OpEquiv:
		return p == q
	default:
		return false
	}
}

// GaussCrossing represents an individual signed crossing in a mathematical knot.
type GaussCrossing struct {
	Index  int  `json:"index"`
	IsOver bool `json:"is_over"`
	Sign   int  `json:"sign"` // +1 for right-handed, -1 for left-handed
}

// KnotDiagram represents a closed 3D knot projection via its extended Gauss Code.
type KnotDiagram struct {
	Name      string          `json:"name"`
	Crossings []GaussCrossing `json:"crossings"`
}

// CrossingNumber returns the minimal crossing count (topological complexity).
func (k *KnotDiagram) CrossingNumber() int {
	return len(k.Crossings) / 2
}

// TrefoilKnot returns the canonical mathematical Trefoil knot (3_1).
func TrefoilKnot() *KnotDiagram {
	return &KnotDiagram{
		Name: "Trefoil (3_1)",
		Crossings: []GaussCrossing{
			{Index: 1, IsOver: true, Sign: 1},
			{Index: 2, IsOver: false, Sign: 1},
			{Index: 3, IsOver: true, Sign: 1},
			{Index: 1, IsOver: false, Sign: 1},
			{Index: 2, IsOver: true, Sign: 1},
			{Index: 3, IsOver: false, Sign: 1},
		},
	}
}
