package verification

import (
	"crypto/ed25519"
	"encoding/hex"
	"errors"
	"math"
	"strconv"
	"strings"
	"time"

	"integin/pkg/id"
)

// Role is the statutory authority of a person in charge of a lifting
// operation (Pillar IV Legal Nexus).
type Role string

const (
	RoleAppointedPerson        Role = "APPOINTED_PERSON"
	RoleLiftDirector           Role = "LIFT_DIRECTOR"
	RoleMarineWarrantySurveyor Role = "MARINE_WARRANTY_SURVEYOR"
	RoleCompetentPerson        Role = "COMPETENT_PERSON"
)

var roleDefaults = map[Role]string{
	RoleAppointedPerson:        "BS 7121",
	RoleLiftDirector:           "ASME P30.1 / OSHA 1926 CC",
	RoleMarineWarrantySurveyor: "DNV-ST-N001 / JRP",
	RoleCompetentPerson:        "LOLER",
}

// criticalUtilizationPercent is the statutory utilization ceiling above which
// a lift is classified critical. Deterministic gate constant (Hazard 7: no
// LLM-derived thresholds).
const criticalUtilizationPercent = 85.0

// MaxTimestampSkew bounds how far ahead of the verifier clock a signed UUIDv7
// timestamp may lie before it is rejected (Hazard 33: forged future stamps).
const MaxTimestampSkew = 5 * time.Minute

var (
	ErrInvalidRole                = errors.New("person-in-charge: invalid role")
	ErrMissingStatutoryStandard   = errors.New("person-in-charge: missing statutory standard")
	ErrMissingTensorPackDigest    = errors.New("person-in-charge: missing proof-witnessed tensor pack digest")
	ErrEmptyENU                   = errors.New("person-in-charge: empty ENU location")
	ErrNonFiniteENU               = errors.New("person-in-charge: non-finite ENU coordinate")
	ErrInvalidTimestamp           = errors.New("person-in-charge: invalid UUIDv7 timestamp")
	ErrFutureTimestamp            = errors.New("person-in-charge: UUIDv7 timestamp lies in the future")
	ErrCriticalLiftRequiresAPOrLD = errors.New("person-in-charge: critical lift requires appointed person or lift director signature")
	ErrInvalidSignature           = errors.New("person-in-charge: invalid signature")
	ErrNoPrivateKey               = errors.New("person-in-charge: signing ceremony has no private key")
	ErrNonFiniteUtilization       = errors.New("person-in-charge: non-finite lift utilization")
)

// Valid reports whether r is a known statutory role.
func (r Role) Valid() bool {
	_, ok := roleDefaults[r]
	return ok
}

// DefaultStatutoryStandard returns the governing regulation for the role.
func (r Role) DefaultStatutoryStandard() string {
	return roleDefaults[r]
}

// CanAuthorizeCritical reports whether the role may cryptographically approve
// a critical lift (utilization > 85% or multi-crane tandem).
func (r Role) CanAuthorizeCritical() bool {
	return r == RoleAppointedPerson || r == RoleLiftDirector
}

// PersonInChargeSignRecord binds a signing authority to the physical facts of
// one lift approval. The cryptographic signature covers exactly:
//
//	Timestamp_UUIDv7 || LocalENU_Coordinates || TensorPackDigest || Role || StatutoryStandard
type PersonInChargeSignRecord struct {
	TimestampUUIDv7   string
	ENUEastMeters     float64
	ENUNorthMeters    float64
	ENUUpMeters       float64
	TensorPackDigest  [32]byte
	Role              Role
	StatutoryStandard string

	// UtilizationPercent and MultiCraneTandem are server-computed lift
	// facts (not client input) used only to gate who may sign.
	UtilizationPercent float64
	MultiCraneTandem   bool
}

// RequiresCriticalRole reports whether this lift legally demands an
// Appointed Person or Lift Director signature.
func (r PersonInChargeSignRecord) RequiresCriticalRole() bool {
	return r.UtilizationPercent > criticalUtilizationPercent || r.MultiCraneTandem
}

// ENUCanonical renders the local ENU coordinates as a deterministic string.
func (r PersonInChargeSignRecord) ENUCanonical() string {
	return strings.Join([]string{
		"E=" + strconv.FormatFloat(r.ENUEastMeters, 'f', 6, 64),
		"N=" + strconv.FormatFloat(r.ENUNorthMeters, 'f', 6, 64),
		"U=" + strconv.FormatFloat(r.ENUUpMeters, 'f', 6, 64),
	}, ",")
}

// SignInput builds the canonical byte stream that is signed and verified.
func (r PersonInChargeSignRecord) SignInput() ([]byte, error) {
	if err := r.validate(time.Now().UTC()); err != nil {
		return nil, err
	}
	return []byte(strings.Join([]string{
		r.TimestampUUIDv7,
		r.ENUCanonical(),
		hex.EncodeToString(r.TensorPackDigest[:]),
		string(r.Role),
		r.StatutoryStandard,
	}, "||")), nil
}

// validate enforces every trust-boundary rule on the record.
func (r PersonInChargeSignRecord) validate(now time.Time) error {
	if !r.Role.Valid() {
		return ErrInvalidRole
	}
	if r.StatutoryStandard == "" {
		return ErrMissingStatutoryStandard
	}
	if r.TensorPackDigest == [32]byte{} {
		return ErrMissingTensorPackDigest
	}
	enu := []float64{r.ENUEastMeters, r.ENUNorthMeters, r.ENUUpMeters}
	nonEmpty := false
	for _, v := range enu {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return ErrNonFiniteENU
		}
		if v != 0 {
			nonEmpty = true
		}
	}
	if !nonEmpty {
		return ErrEmptyENU
	}
	if !id.IsValidV7(r.TimestampUUIDv7) {
		return ErrInvalidTimestamp
	}
	ts, err := id.ParseTime(r.TimestampUUIDv7)
	if err != nil {
		return ErrInvalidTimestamp
	}
	if ts.After(now.Add(MaxTimestampSkew)) {
		return ErrFutureTimestamp
	}
	return nil
}

// SigningCeremony holds the private key that authorizes lift approvals.
type SigningCeremony struct {
	PrivateKey ed25519.PrivateKey
}

// Sign enforces the role-based critical-lift rule, mints a fresh RFC 9562
// UUIDv7 timestamp (Hazard 33: never signs a client-supplied clock), and
// returns the Ed25519 signature over the canonical record input.
func (c *SigningCeremony) Sign(rec *PersonInChargeSignRecord) ([]byte, error) {
	if c == nil || c.PrivateKey == nil {
		return nil, ErrNoPrivateKey
	}
	if rec.RequiresCriticalRole() && !rec.Role.CanAuthorizeCritical() {
		return nil, ErrCriticalLiftRequiresAPOrLD
	}
	if rec.StatutoryStandard == "" {
		rec.StatutoryStandard = rec.Role.DefaultStatutoryStandard()
	}
	if math.IsNaN(rec.UtilizationPercent) || math.IsInf(rec.UtilizationPercent, 0) {
		return nil, ErrNonFiniteUtilization
	}
	ts, err := id.NewV7()
	if err != nil {
		return nil, err
	}
	rec.TimestampUUIDv7 = ts
	input, err := rec.SignInput()
	if err != nil {
		return nil, err
	}
	return ed25519.Sign(c.PrivateKey, input), nil
}

// VerifyPersonInChargeSignature verifies the Ed25519 signature over the
// canonical record input, the UUIDv7 timestamp validity, a non-empty ENU
// location, the presence of the proof-witnessed tensor pack digest, and the
// role authority required for critical lifts.
func VerifyPersonInChargeSignature(rec PersonInChargeSignRecord, pub ed25519.PublicKey, signature []byte) error {
	if len(signature) != ed25519.SignatureSize {
		return ErrInvalidSignature
	}
	if err := rec.validate(time.Now().UTC()); err != nil {
		return err
	}
	if rec.RequiresCriticalRole() && !rec.Role.CanAuthorizeCritical() {
		return ErrCriticalLiftRequiresAPOrLD
	}
	input, err := rec.SignInput()
	if err != nil {
		return err
	}
	if !ed25519.Verify(pub, input, signature) {
		return ErrInvalidSignature
	}
	return nil
}
