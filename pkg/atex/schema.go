package atex

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"
)

var (
	ErrZoneViolation         = errors.New("atex: equipment not certified for deployment in target hazardous zone")
	ErrVoltageBreach         = errors.New("atex: barrier open-circuit voltage Uo exceeds apparatus input voltage Ui")
	ErrCurrentBreach         = errors.New("atex: barrier short-circuit current Io exceeds apparatus input current Ii")
	ErrPowerBreach           = errors.New("atex: barrier max output power Po exceeds apparatus input power Pi")
	ErrCapacitanceBreach     = errors.New("atex: lumped capacitance (Ci + Ccable) exceeds barrier permissible Co")
	ErrInductanceBreach      = errors.New("atex: lumped inductance (Li + Lcable) exceeds barrier permissible Lo")
	ErrTemperatureExceeded   = errors.New("atex: apparatus maximum surface temperature exceeds explosive gas ignition point")
	ErrInvalidGasGroup       = errors.New("atex: apparatus gas group is incompatible with ambient atmosphere")
	ErrZeroApparatusDID      = errors.New("atex: apparatus DID cannot be empty")
	ErrZeroBarrierDID        = errors.New("atex: safety barrier DID cannot be empty")
)

type HazardousZone string

const (
	Zone0 HazardousZone = "ZONE_0" // Continuous explosive gas/vapor presence (>1000 hrs/yr)
	Zone1 HazardousZone = "ZONE_1" // Likely explosive gas presence under normal operation (10-1000 hrs/yr)
	Zone2 HazardousZone = "ZONE_2" // Unlikely or short-duration explosive gas presence (<10 hrs/yr)
	Safe  HazardousZone = "SAFE_ZONE"
)

type GasGroup string

const (
	GroupIIA GasGroup = "IIA" // Propane, methane (least ignitable)
	GroupIIB GasGroup = "IIB" // Ethylene
	GroupIIC GasGroup = "IIC" // Hydrogen, acetylene (most ignitable)
)

type TempClass string

const (
	T1 TempClass = "T1" // 450 C
	T2 TempClass = "T2" // 300 C
	T3 TempClass = "T3" // 200 C
	T4 TempClass = "T4" // 135 C
	T5 TempClass = "T5" // 100 C
	T6 TempClass = "T6" // 85 C (strictest)
)

// TempClassMaxTemp maps IEC 60079-0 temperature classes to maximum permissible surface temperatures.
func TempClassMaxTemp(tc TempClass) float64 {
	switch tc {
	case T1:
		return 450.0
	case T2:
		return 300.0
	case T3:
		return 200.0
	case T4:
		return 135.0
	case T5:
		return 100.0
	case T6:
		return 85.0
	default:
		return 85.0
	}
}

// IntrinsicallySafeParameters defines entity parameters per IEC/EN 60079-11.
type IntrinsicallySafeParameters struct {
	Ui float64 `json:"ui"` // Volts (max input voltage apparatus can withstand)
	Ii float64 `json:"ii"` // Amps (max input current apparatus can withstand)
	Pi float64 `json:"pi"` // Watts (max input power apparatus can dissipate)
	Ci float64 `json:"ci"` // uF (internal unprotected capacitance)
	Li float64 `json:"li"` // uH (internal unprotected inductance)
}

// BarrierParameters defines safety barrier electrical output boundaries per IEC 60079-11.
type BarrierParameters struct {
	Uo float64 `json:"uo"` // Volts (max output voltage from barrier)
	Io float64 `json:"io"` // Amps (max output current from barrier)
	Po float64 `json:"po"` // Watts (max output power from barrier)
	Co float64 `json:"co"` // uF (max external capacitance barrier can drive)
	Lo float64 `json:"lo"` // uH (max external inductance barrier can drive)
}

// CableParameters represents lumped interconnecting cable parameters.
type CableParameters struct {
	LengthMeters float64 `json:"length_meters"`
	CcablePerM   float64 `json:"ccable_per_m"` // uF/m
	LcablePerM   float64 `json:"lcable_per_m"` // uH/m
}

// IntrinsicSafetyLoop verifies compatibility between safety barrier, cable, and field device.
type IntrinsicSafetyLoop struct {
	ApparatusDID    string                      `json:"apparatus_did"`
	BarrierDID      string                      `json:"barrier_did"`
	Zone            HazardousZone               `json:"zone"`
	EnvironmentGas  GasGroup                    `json:"environment_gas"`
	ApparatusGas    GasGroup                    `json:"apparatus_gas"`
	ApparatusTemp   TempClass                   `json:"apparatus_temp"`
	AutoIgnitionC   float64                     `json:"auto_ignition_c"` // Explosive mixture auto-ignition temperature
	ApparatusParams IntrinsicallySafeParameters `json:"apparatus_params"`
	BarrierParams   BarrierParameters           `json:"barrier_params"`
	Cable           CableParameters             `json:"cable"`
}

// VerificationResult encapsulates mathematical compliance proof for hazardous deployment.
type VerificationResult struct {
	IsCompliant     bool      `json:"is_compliant"`
	TotalCi         float64   `json:"total_ci"`
	TotalLi         float64   `json:"total_li"`
	EvaluatedAt     time.Time `json:"evaluated_at"`
	IsolationStatus string    `json:"isolation_status"`
	FailureReason   string    `json:"failure_reason,omitempty"`
}

// ValidateIntrinsicSafetyLoop executes strict IEC/EN 60079-11 entity comparison rules.
func ValidateIntrinsicSafetyLoop(loop IntrinsicSafetyLoop) (VerificationResult, error) {
	if loop.ApparatusDID == "" {
		return VerificationResult{}, ErrZeroApparatusDID
	}
	if loop.BarrierDID == "" {
		return VerificationResult{}, ErrZeroBarrierDID
	}

	// 1. Gas group hierarchy validation: IIC satisfies IIB & IIA; IIB satisfies IIA.
	if !isGasGroupCompatible(loop.ApparatusGas, loop.EnvironmentGas) {
		return VerificationResult{
			IsCompliant:     false,
			IsolationStatus: "ISOLATION_TRIGGERED",
			FailureReason:   ErrInvalidGasGroup.Error(),
			EvaluatedAt:     time.Now().UTC(),
		}, ErrInvalidGasGroup
	}

	// 2. Thermal classification safety check (IEC 60079-0)
	maxSurfaceTemp := TempClassMaxTemp(loop.ApparatusTemp)
	if loop.AutoIgnitionC > 0 && maxSurfaceTemp >= loop.AutoIgnitionC {
		return VerificationResult{
			IsCompliant:     false,
			IsolationStatus: "THERMAL_LOCKOUT",
			FailureReason:   ErrTemperatureExceeded.Error(),
			EvaluatedAt:     time.Now().UTC(),
		}, ErrTemperatureExceeded
	}

	// 3. Electrical Entity Parameter comparison:
	// Ui >= Uo
	if loop.ApparatusParams.Ui < loop.BarrierParams.Uo {
		return VerificationResult{
			IsCompliant:     false,
			IsolationStatus: "VOLTAGE_LOCKOUT",
			FailureReason:   ErrVoltageBreach.Error(),
			EvaluatedAt:     time.Now().UTC(),
		}, ErrVoltageBreach
	}

	// Ii >= Io
	if loop.ApparatusParams.Ii < loop.BarrierParams.Io {
		return VerificationResult{
			IsCompliant:     false,
			IsolationStatus: "CURRENT_LOCKOUT",
			FailureReason:   ErrCurrentBreach.Error(),
			EvaluatedAt:     time.Now().UTC(),
		}, ErrCurrentBreach
	}

	// Pi >= Po
	if loop.ApparatusParams.Pi < loop.BarrierParams.Po {
		return VerificationResult{
			IsCompliant:     false,
			IsolationStatus: "POWER_LOCKOUT",
			FailureReason:   ErrPowerBreach.Error(),
			EvaluatedAt:     time.Now().UTC(),
		}, ErrPowerBreach
	}

	// 4. Lumped Reactive Parameters (Capacitance & Inductance):
	totalCableC := loop.Cable.LengthMeters * loop.Cable.CcablePerM
	totalC := loop.ApparatusParams.Ci + totalCableC
	if totalC > loop.BarrierParams.Co {
		return VerificationResult{
			IsCompliant:     false,
			TotalCi:         totalC,
			IsolationStatus: "CAPACITANCE_LOCKOUT",
			FailureReason:   ErrCapacitanceBreach.Error(),
			EvaluatedAt:     time.Now().UTC(),
		}, ErrCapacitanceBreach
	}

	totalCableL := loop.Cable.LengthMeters * loop.Cable.LcablePerM
	totalL := loop.ApparatusParams.Li + totalCableL
	if totalL > loop.BarrierParams.Lo {
		return VerificationResult{
			IsCompliant:     false,
			TotalLi:         totalL,
			IsolationStatus: "INDUCTANCE_LOCKOUT",
			FailureReason:   ErrInductanceBreach.Error(),
			EvaluatedAt:     time.Now().UTC(),
		}, ErrInductanceBreach
	}

	return VerificationResult{
		IsCompliant:     true,
		TotalCi:         totalC,
		TotalLi:         totalL,
		IsolationStatus: "GALVANICALLY_SECURED",
		EvaluatedAt:     time.Now().UTC(),
	}, nil
}

func isGasGroupCompatible(apparatus GasGroup, ambient GasGroup) bool {
	if apparatus == GroupIIC {
		return true // Group IIC apparatus is safe for IIC, IIB, and IIA
	}
	if apparatus == GroupIIB && (ambient == GroupIIB || ambient == GroupIIA) {
		return true
	}
	if apparatus == GroupIIA && ambient == GroupIIA {
		return true
	}
	return false
}

// CertifiedZonePass represents an Ed25519-signed authorization certificate for Ex deployment.
type CertifiedZonePass struct {
	PassDID        string    `json:"pass_did"` // did:integin:expass:<uuid>
	ApparatusDID   string    `json:"apparatus_did"`
	BarrierDID     string    `json:"barrier_did"`
	Zone           string    `json:"zone"`
	GasGroup       string    `json:"gas_group"`
	TempClass      string    `json:"temp_class"`
	IssuedAt       time.Time `json:"issued_at"`
	ExpiresAt      time.Time `json:"expires_at"`
	Digest         string    `json:"digest"`
	SignatureHex   string    `json:"signature_hex"`
}

// IssueZonePass signs a cryptographic permit allowing tool operation in an ATEX zone.
func IssueZonePass(loop IntrinsicSafetyLoop, privKey ed25519.PrivateKey, validity time.Duration) (*CertifiedZonePass, error) {
	res, err := ValidateIntrinsicSafetyLoop(loop)
	if err != nil || !res.IsCompliant {
		return nil, fmt.Errorf("atex: loop failed validation: %w", err)
	}

	var rawUUID [16]byte
	if _, err := rand.Read(rawUUID[:]); err != nil {
		return nil, err
	}
	passDID := fmt.Sprintf("did:integin:expass:%s", hex.EncodeToString(rawUUID[:]))

	now := time.Now().UTC()
	pass := &CertifiedZonePass{
		PassDID:      passDID,
		ApparatusDID: loop.ApparatusDID,
		BarrierDID:   loop.BarrierDID,
		Zone:         string(loop.Zone),
		GasGroup:     string(loop.EnvironmentGas),
		TempClass:    string(loop.ApparatusTemp),
		IssuedAt:     now,
		ExpiresAt:    now.Add(validity),
	}

	payload := fmt.Sprintf("%s|%s|%s|%s|%s|%s|%d|%d",
		pass.PassDID, pass.ApparatusDID, pass.BarrierDID,
		pass.Zone, pass.GasGroup, pass.TempClass,
		pass.IssuedAt.Unix(), pass.ExpiresAt.Unix(),
	)
	digest := sha256.Sum256([]byte(payload))
	pass.Digest = hex.EncodeToString(digest[:])

	sig := ed25519.Sign(privKey, digest[:])
	pass.SignatureHex = hex.EncodeToString(sig)

	return pass, nil
}
