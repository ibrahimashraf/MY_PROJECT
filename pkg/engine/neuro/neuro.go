package neuro

import "math"

// HodgkinHuxleyNeuron models membrane potential dynamics.
type HodgkinHuxleyNeuron struct {
	V    float64 // Membrane potential (mV)
	M, H float64 // Na+ gate variables
	N    float64 // K+ gate variable
}

// DefaultHodgkinHuxley returns a resting state neuron.
func DefaultHodgkinHuxley() HodgkinHuxleyNeuron {
	return HodgkinHuxleyNeuron{V: -65.0, M: 0.05, H: 0.6, N: 0.32}
}

// alphaN, betaN etc are standard HH rate functions
func alphaN(v float64) float64 { return 0.01 * (v + 55.0) / (1.0 - math.Exp(-(v+55.0)/10.0)) }
func betaN(v float64) float64  { return 0.125 * math.Exp(-(v+65.0)/80.0) }
func alphaM(v float64) float64 { return 0.1 * (v + 40.0) / (1.0 - math.Exp(-(v+40.0)/10.0)) }
func betaM(v float64) float64  { return 4.0 * math.Exp(-(v+65.0)/18.0) }
func alphaH(v float64) float64 { return 0.07 * math.Exp(-(v+65.0)/20.0) }
func betaH(v float64) float64  { return 1.0 / (1.0 + math.Exp(-(v+35.0)/10.0)) }

// Step advances HH neuron by dt milliseconds given external current input (uA/cm^2).
func (n HodgkinHuxleyNeuron) Step(dt, iExt float64) HodgkinHuxleyNeuron {
	const gNa, gK, gL = 120.0, 36.0, 0.3
	const eNa, eK, eL = 50.0, -77.0, -54.387
	const Cm = 1.0

	iNa := gNa * n.M * n.M * n.M * n.H * (n.V - eNa)
	iK := gK * n.N * n.N * n.N * n.N * (n.V - eK)
	iL := gL * (n.V - eL)

	dV := (iExt - iNa - iK - iL) / Cm
	dM := alphaM(n.V)*(1.0-n.M) - betaM(n.V)*n.M
	dH := alphaH(n.V)*(1.0-n.H) - betaH(n.V)*n.H
	dN := alphaN(n.V)*(1.0-n.N) - betaN(n.V)*n.N

	return HodgkinHuxleyNeuron{
		V: n.V + dt*dV,
		M: math.Max(0, math.Min(1, n.M+dt*dM)),
		H: math.Max(0, math.Min(1, n.H+dt*dH)),
		N: math.Max(0, math.Min(1, n.N+dt*dN)),
	}
}

// SynapticWeight models Hebbian STDP: dW = A+ * exp(-dt/tau+) if pre before post
func STDPWeightUpdate(weight, deltaTms, aPlus, aMinus, tauPlus, tauMinus float64) float64 {
	var dw float64
	if deltaTms > 0 {
		dw = aPlus * math.Exp(-deltaTms/tauPlus)
	} else {
		dw = -aMinus * math.Exp(deltaTms/tauMinus)
	}
	return math.Max(0, math.Min(1, weight+dw))
}
