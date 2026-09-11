package fft

import (
	"errors"
	"math"
	"math/cmplx"
)

// FFT computes the discrete Fourier transform of a sequence using Radix-2 Cooley-Tukey.
// Input length must be a power of 2.
func FFT(x []complex128) ([]complex128, error) {
	n := len(x)
	if n == 0 {
		return nil, errors.New("input slice cannot be empty")
	}
	if n&(n-1) != 0 {
		return nil, errors.New("input length must be a power of 2 for Radix-2 FFT")
	}

	out := make([]complex128, n)
	copy(out, x)
	bitReverse(out)

	for s := 1; s <= int(math.Log2(float64(n))); s++ {
		m := 1 << s
		m2 := m >> 1
		wM := cmplx.Exp(complex(0, -2.0*math.Pi/float64(m)))

		for k := 0; k < n; k += m {
			w := complex(1, 0)
			for j := 0; j < m2; j++ {
				t := w * out[k+j+m2]
				u := out[k+j]
				out[k+j] = u + t
				out[k+j+m2] = u - t
				w *= wM
			}
		}
	}

	return out, nil
}

// IFFT computes the inverse discrete Fourier transform.
func IFFT(x []complex128) ([]complex128, error) {
	n := len(x)
	if n == 0 {
		return nil, errors.New("input slice cannot be empty")
	}

	// Conjugate input
	conj := make([]complex128, n)
	for i, v := range x {
		conj[i] = cmplx.Conj(v)
	}

	// Forward FFT on conjugate
	fwd, err := FFT(conj)
	if err != nil {
		return nil, err
	}

	// Conjugate and divide by N
	invN := complex(1.0/float64(n), 0)
	for i, v := range fwd {
		fwd[i] = cmplx.Conj(v) * invN
	}

	return fwd, nil
}

// PowerSpectralDensity computes the squared magnitude spectrum |X[k]|^2 / N.
func PowerSpectralDensity(fftResult []complex128) []float64 {
	n := len(fftResult)
	if n == 0 {
		return nil
	}
	psd := make([]float64, n/2+1)
	invN := 1.0 / float64(n)

	for i := 0; i <= n/2; i++ {
		mag := cmplx.Abs(fftResult[i])
		psd[i] = (mag * mag) * invN
	}

	return psd
}

func bitReverse(a []complex128) {
	n := len(a)
	j := 0
	for i := 0; i < n-1; i++ {
		if i < j {
			a[i], a[j] = a[j], a[i]
		}
		k := n >> 1
		for k <= j {
			j -= k
			k >>= 1
		}
		j += k
	}
}
