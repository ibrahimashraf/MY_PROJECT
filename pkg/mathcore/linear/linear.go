package linear

import (
	"errors"
	"math"
)

// Matrix represents a 2D dense float64 matrix (Row-major: rows x cols).
type Matrix struct {
	Rows int
	Cols int
	Data []float64
}

// NewMatrix allocates a zero-initialized matrix.
func NewMatrix(rows, cols int) *Matrix {
	return &Matrix{
		Rows: rows,
		Cols: cols,
		Data: make([]float64, rows*cols),
	}
}

// NewMatrixFromData creates a matrix from an existing slice.
func NewMatrixFromData(rows, cols int, data []float64) (*Matrix, error) {
	if len(data) != rows*cols {
		return nil, errors.New("data slice length does not match rows*cols")
	}
	cpy := make([]float64, len(data))
	copy(cpy, data)
	return &Matrix{Rows: rows, Cols: cols, Data: cpy}, nil
}

// Get returns element at (r, c).
func (m *Matrix) Get(r, c int) float64 {
	return m.Data[r*m.Cols+c]
}

// Set sets element at (r, c).
func (m *Matrix) Set(r, c int, val float64) {
	m.Data[r*m.Cols+c] = val
}

// Mul computes matrix multiplication: m * o.
func (m *Matrix) Mul(o *Matrix) (*Matrix, error) {
	if m.Cols != o.Rows {
		return nil, errors.New("matrix dimension mismatch for multiplication")
	}
	res := NewMatrix(m.Rows, o.Cols)
	for r := 0; r < m.Rows; r++ {
		for c := 0; c < o.Cols; c++ {
			sum := 0.0
			for k := 0; k < m.Cols; k++ {
				sum += m.Data[r*m.Cols+k] * o.Data[k*o.Cols+c]
			}
			res.Data[r*res.Cols+c] = sum
		}
	}
	return res, nil
}

// Transpose returns a new transposed matrix.
func (m *Matrix) Transpose() *Matrix {
	res := NewMatrix(m.Cols, m.Rows)
	for r := 0; r < m.Rows; r++ {
		for c := 0; c < m.Cols; c++ {
			res.Data[c*res.Cols+r] = m.Data[r*m.Cols+c]
		}
	}
	return res
}

// Determinant computes the matrix determinant using Gaussian elimination with partial pivoting.
func (m *Matrix) Determinant() (float64, error) {
	if m.Rows != m.Cols {
		return 0, errors.New("matrix must be square for determinant")
	}
	n := m.Rows
	a := make([]float64, len(m.Data))
	copy(a, m.Data)

	det := 1.0
	for i := 0; i < n; i++ {
		// Pivot selection
		maxRow := i
		maxVal := math.Abs(a[i*n+i])
		for k := i + 1; k < n; k++ {
			if math.Abs(a[k*n+i]) > maxVal {
				maxVal = math.Abs(a[k*n+i])
				maxRow = k
			}
		}

		if maxVal < 1e-12 {
			return 0, nil // Singular matrix
		}

		if maxRow != i {
			for c := 0; c < n; c++ {
				a[i*n+c], a[maxRow*n+c] = a[maxRow*n+c], a[i*n+c]
			}
			det = -det
		}

		pivot := a[i*n+i]
		det *= pivot

		for k := i + 1; k < n; k++ {
			factor := a[k*n+i] / pivot
			for c := i; c < n; c++ {
				a[k*n+c] -= factor * a[i*n+c]
			}
		}
	}

	return det, nil
}

// SolveLinear solves Ax = b using Gaussian elimination with back-substitution.
func SolveLinear(a *Matrix, b []float64) ([]float64, error) {
	if a.Rows != a.Cols || a.Rows != len(b) {
		return nil, errors.New("dimension mismatch in SolveLinear")
	}
	n := a.Rows
	m := make([]float64, n*(n+1))

	// Augmented matrix [A | b]
	for r := 0; r < n; r++ {
		for c := 0; c < n; c++ {
			m[r*(n+1)+c] = a.Get(r, c)
		}
		m[r*(n+1)+n] = b[r]
	}

	// Forward elimination
	for i := 0; i < n; i++ {
		pivot := m[i*(n+1)+i]
		if math.Abs(pivot) < 1e-12 {
			return nil, errors.New("singular matrix in linear solver")
		}

		for k := i + 1; k < n; k++ {
			factor := m[k*(n+1)+i] / pivot
			for c := i; c <= n; c++ {
				m[k*(n+1)+c] -= factor * m[i*(n+1)+c]
			}
		}
	}

	// Back substitution
	x := make([]float64, n)
	for i := n - 1; i >= 0; i-- {
		sum := m[i*(n+1)+n]
		for j := i + 1; j < n; j++ {
			sum -= m[i*(n+1)+j] * x[j]
		}
		x[i] = sum / m[i*(n+1)+i]
	}

	return x, nil
}
