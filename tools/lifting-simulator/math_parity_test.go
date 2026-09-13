// Package lifting_simulator cross-validates the maths embedded in
// tools/lifting-simulator (2D sling rigging and 3D tandem ground bearing)
// against the authoritative pkg/rulesengine implementations.
package lifting_simulator

import (
	"math"
	"testing"

	"integin/pkg/rulesengine"
)

const (
	slingLoadT      = 10.0 // 2D vector demo load (t)
	simLoadT        = 45.0 // 3D tandem vessel mass (t)
	simSplitToCrane = 0.52 // fraction of load carried by crane 1
	simCraneSelfT   = 60.0 // crane dead weight on 4 outriggers (t)
	simPads         = 4
	simBearingAreaM = 2.25 // outrigger mat area (m^2)
	simAllowableKPa = 220.0
	simFosPass      = 1.5 // PASS threshold implemented in app.js
)

// slingTension2D is the analysis mirror of app.js / rigging.go:
//
//	T = load / (2 * sin(angle))
func slingTension2D(loadT, angleDeg float64) float64 {
	return loadT / (2 * math.Sin(angleDeg*math.Pi/180))
}

// tandemPadLoad is the analysis mirror of app.js updateKinematics: the load
// fraction carried by one crane plus its self weight, spread over its pads.
func tandemPadLoad(loadT, split, craneSelfT float64, pads int) float64 {
	return (loadT*split + craneSelfT) / float64(pads)
}

func bearingKPa(padLoadT, areaM2 float64) float64 {
	return padLoadT * 9.80665 / areaM2
}

func bearingFoS(allowable, actualKPa float64) float64 {
	return allowable / actualKPa
}

func almostEqual(a, b, tol float64) bool {
	return math.Abs(a-b) <= tol
}

func TestSlingTensionParity(t *testing.T) {
	for _, angle := range []float64{30, 45, 60, 75, 89} {
		want := slingTension2D(slingLoadT, angle)
		got, passed, lockout, err := rulesengine.SlingTension2Leg(slingLoadT, angle)
		if err != nil || !passed || lockout {
			t.Fatalf("angle %g: SlingTension2Leg = (%v, %v, %v, %v)", angle, got, passed, lockout, err)
		}
		if !almostEqual(got, want, 1e-9) {
			t.Errorf("angle %g: got %.9f, analysis mirror wants %.9f", angle, got, want)
		}
	}
}

func TestSlingCriticalAngleThreshold(t *testing.T) {
	if _, _, lockout, err := rulesengine.SlingTension2Leg(slingLoadT, 30); err != nil {
		t.Fatalf("30 deg must be permitted: %v", err)
	} else if lockout {
		t.Fatal("30 deg must not lock out")
	}
	if v := slingTension2D(slingLoadT, 30); v < slingLoadT {
		t.Fatalf("half-lifted 30 deg tension %g must exceed load %g", v, slingLoadT)
	}
	if _, _, lockout, err := rulesengine.SlingTension2Leg(slingLoadT, 29.9); err == nil || !lockout {
		t.Fatal("29.9 deg must lock out")
	}
	if v := slingTension2D(slingLoadT, 29.9); v <= slingTension2D(slingLoadT, 30) {
		t.Fatal("shallower angle must not yield lower tension")
	}
}

func TestTandemGroundBearingParity(t *testing.T) {
	padLoad := tandemPadLoad(simLoadT, simSplitToCrane, simCraneSelfT, simPads)
	wantKPa := bearingKPa(padLoad, simBearingAreaM)
	gotKPa, ok, err := rulesengine.GroundBearingPressure(padLoad, simBearingAreaM)
	if err != nil {
		t.Fatalf("GroundBearingPressure: %v", err)
	}
	if !ok {
		t.Fatal("simulator scenario must yield a positive pressure")
	}
	if !almostEqual(gotKPa, wantKPa, 1e-9) {
		t.Errorf("got %.9f kPa, analysis mirror wants %.9f kPa", gotKPa, wantKPa)
	}

	fos := bearingFoS(simAllowableKPa, gotKPa)
	if fos < simFosPass {
		t.Fatalf("default scenario FoS = %.3f, must be >= %.1f to PASS", fos, simFosPass)
	}
}

func TestTandemBearingThreshold(t *testing.T) {
	fos := bearingFoS(simAllowableKPa, bearingKPa(tandemPadLoad(simLoadT, simSplitToCrane, simCraneSelfT, simPads), simBearingAreaM))
	if math.IsNaN(fos) || math.IsInf(fos, 0) || fos < simFosPass {
		t.Fatalf("simulator PASS threshold violated: FoS = %v", fos)
	}
	// Hold the pad area steady and inflate the load until the same formula
	// just fails the 1.5 threshold: the verdict must flip exactly there.
	failLoad := simLoadT
	for bearingFoS(simAllowableKPa, bearingKPa(tandemPadLoad(failLoad, simSplitToCrane, simCraneSelfT, simPads), simBearingAreaM)) >= simFosPass {
		failLoad++
		if failLoad > 1000 {
			t.Fatal("threshold never flips")
		}
	}
	if bearingFoS(simAllowableKPa, bearingKPa(tandemPadLoad(failLoad-1, simSplitToCrane, simCraneSelfT, simPads), simBearingAreaM)) < simFosPass {
		t.Fatal("previous load must still pass")
	}
	// Zero area must be refused by the authoritative engine.
	if _, _, err := rulesengine.GroundBearingPressure(tandemPadLoad(simLoadT, simSplitToCrane, simCraneSelfT, simPads), 0); err == nil {
		t.Fatal("zero bearing area must be rejected")
	}
}
