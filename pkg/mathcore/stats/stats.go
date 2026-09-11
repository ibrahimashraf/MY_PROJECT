package stats

import (
	"errors"
	"math"
)

// Mean calculates arithmetic average of slice x.
func Mean(x []float64) float64 {
	if len(x) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range x {
		sum += v
	}
	return sum / float64(len(x))
}

// Variance calculates sample variance (N-1 denominator).
func Variance(x []float64) float64 {
	n := len(x)
	if n < 2 {
		return 0
	}
	m := Mean(x)
	sumSq := 0.0
	for _, v := range x {
		d := v - m
		sumSq += d * d
	}
	return sumSq / float64(n-1)
}

// StdDev calculates sample standard deviation.
func StdDev(x []float64) float64 {
	return math.Sqrt(Variance(x))
}

// Covariance computes sample covariance between x and y.
func Covariance(x, y []float64) (float64, error) {
	n := len(x)
	if n != len(y) {
		return 0, errors.New("length of x and y must be identical")
	}
	if n < 2 {
		return 0, errors.New("sample size must be >= 2")
	}

	mx := Mean(x)
	my := Mean(y)

	cov := 0.0
	for i := 0; i < n; i++ {
		cov += (x[i] - mx) * (y[i] - my)
	}

	return cov / float64(n-1), nil
}

// Correlation computes Pearson correlation coefficient r in [-1.0, 1.0].
func Correlation(x, y []float64) (float64, error) {
	cov, err := Covariance(x, y)
	if err != nil {
		return 0, err
	}
	sx := StdDev(x)
	sy := StdDev(y)

	if sx < 1e-12 || sy < 1e-12 {
		return 0, nil // Zero variance
	}

	return cov / (sx * sy), nil
}

// OLSResult contains parameters of an Ordinary Least Squares regression: y = beta*x + alpha.
type OLSResult struct {
	Alpha float64 `json:"alpha"` // Intercept
	Beta  float64 `json:"beta"`  // Slope
	R2    float64 `json:"r2"`    // Coefficient of determination
	RMSE  float64 `json:"rmse"`  // Root Mean Square Error
}

// LinearRegression computes univariate OLS estimates.
func LinearRegression(x, y []float64) (*OLSResult, error) {
	n := len(x)
	if n != len(y) || n < 2 {
		return nil, errors.New("insufficient data points for linear regression")
	}

	cov, err := Covariance(x, y)
	if err != nil {
		return nil, err
	}
	varX := Variance(x)
	if varX < 1e-12 {
		return nil, errors.New("variance of x is zero")
	}

	beta := cov / varX
	alpha := Mean(y) - beta*Mean(x)

	// Compute R^2 and RMSE
	ssTot := 0.0
	ssRes := 0.0
	my := Mean(y)

	for i := 0; i < n; i++ {
		yPred := beta*x[i] + alpha
		res := y[i] - yPred
		ssRes += res * res
		d := y[i] - my
		ssTot += d * d
	}

	r2 := 0.0
	if ssTot > 1e-12 {
		r2 = 1.0 - (ssRes / ssTot)
	}
	rmse := math.Sqrt(ssRes / float64(n))

	return &OLSResult{
		Alpha: alpha,
		Beta:  beta,
		R2:    r2,
		RMSE:  rmse,
	}, nil
}

// AR1Model represents an autoregressive time-series model: y_t = c + phi*y_{t-1} + e_t.
type AR1Model struct {
	Constant float64 `json:"constant"`
	Phi      float64 `json:"phi"`
	Sigma2   float64 `json:"sigma2"`
}

// FitAR1 estimates AR(1) coefficients via conditional least squares.
func FitAR1(series []float64) (*AR1Model, error) {
	n := len(series)
	if n < 4 {
		return nil, errors.New("series too short for AR(1) estimation")
	}

	y := series[1:]
	x := series[:n-1]

	reg, err := LinearRegression(x, y)
	if err != nil {
		return nil, err
	}

	return &AR1Model{
		Constant: reg.Alpha,
		Phi:      reg.Beta,
		Sigma2:   reg.RMSE * reg.RMSE,
	}, nil
}
