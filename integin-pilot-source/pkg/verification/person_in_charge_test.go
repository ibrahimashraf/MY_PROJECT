package verification

import (
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"fmt"
	"math"
	"testing"
	"time"

	"integin/pkg/id"
)

func signRec(t *testing.T, priv ed25519.PrivateKey, rec *PersonInChargeSignRecord) []byte {
	t.Helper()
	sig, err := (&SigningCeremony{PrivateKey: priv}).Sign(rec)
	if err != nil {
		t.Fatalf("Sign failed: %v", err)
	}
	return sig
}

func TestSigningCeremonySignAndVerifyPerRole(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	for _, role := range []Role{RoleAppointedPerson, RoleLiftDirector, RoleMarineWarrantySurveyor, RoleCompetentPerson} {
		rec := PersonInChargeSignRecord{
			ENUEastMeters:    12.5,
			ENUNorthMeters:   -3.25,
			ENUUpMeters:      7.0,
			TensorPackDigest: [32]byte{1, 2, 3, 4},
			Role:             role,
		}
		sig := signRec(t, priv, &rec)
		if len(sig) != ed25519.SignatureSize {
			t.Fatalf("role %s: bad signature length %d", role, len(sig))
		}
		if !id.IsValidV7(rec.TimestampUUIDv7) {
			t.Fatalf("role %s: timestamp is not UUIDv7", role)
		}
		if rec.StatutoryStandard == "" {
			t.Fatalf("role %s: statutory standard not defaulted", role)
		}
		if err := VerifyPersonInChargeSignature(rec, pub, sig); err != nil {
			t.Fatalf("role %s: verify failed: %v", role, err)
		}
	}
}

func TestRoleCanonicalStandards(t *testing.T) {
	got := RoleAppointedPerson.DefaultStatutoryStandard()
	if got != "BS 7121" {
		t.Fatalf("appointed person standard = %q", got)
	}
	if !RoleAppointedPerson.Valid() || !RoleLiftDirector.Valid() ||
		!RoleMarineWarrantySurveyor.Valid() || !RoleCompetentPerson.Valid() {
		t.Fatal("expected all four roles valid")
	}
	if Role("CORPORATE_CEO").Valid() {
		t.Fatal("expected unknown role to be invalid")
	}
}

func TestCriticalLiftAuthority(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	base := PersonInChargeSignRecord{
		ENUEastMeters:    1.0,
		ENUNorthMeters:   2.0,
		ENUUpMeters:      3.0,
		TensorPackDigest: [32]byte{9},
	}
	critical := []PersonInChargeSignRecord{
		LiftCriticalityUtilization(base, 85.1),
		LiftCriticalityUtilization(base, 99.9),
		MultiCrane(base),
	}
	for _, rec := range critical {
		for _, role := range []Role{RoleMarineWarrantySurveyor, RoleCompetentPerson} {
			rec.Role = role
			rec.StatutoryStandard = role.DefaultStatutoryStandard()
			if !rec.RequiresCriticalRole() {
				t.Fatalf("expected critical lift for %s", role)
			}
			if _, err := (&SigningCeremony{PrivateKey: priv}).Sign(&rec); err != ErrCriticalLiftRequiresAPOrLD {
				t.Fatalf("role %s: expected critical-role denial, got %v", role, err)
			}
		}
	}
	authorized := []Role{RoleAppointedPerson, RoleLiftDirector}
	for _, rec := range critical {
		for _, role := range authorized {
			rec.Role = role
			rec.StatutoryStandard = role.DefaultStatutoryStandard()
			if sig := signRec(t, priv, &rec); VerifyPersonInChargeSignature(rec, pub, sig) != nil {
				t.Fatalf("authorized role %s rejected on critical lift", role)
			}
		}
	}
}

func LiftCriticalityUtilization(base PersonInChargeSignRecord, pct float64) PersonInChargeSignRecord {
	r := base
	r.UtilizationPercent = pct
	return r
}

func MultiCrane(base PersonInChargeSignRecord) PersonInChargeSignRecord {
	r := base
	r.MultiCraneTandem = true
	return r
}

func TestCriticalBoundaryAt85Exact(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	rec := PersonInChargeSignRecord{
		ENUEastMeters:      1.0,
		ENUNorthMeters:     2.0,
		ENUUpMeters:        3.0,
		TensorPackDigest:   [32]byte{5},
		Role:               RoleCompetentPerson,
		UtilizationPercent: 85.0,
	}
	if rec.RequiresCriticalRole() {
		t.Fatal("exactly 85% must not be critical")
	}
	rec.StatutoryStandard = rec.Role.DefaultStatutoryStandard()
	sig := signRec(t, priv, &rec)
	if err := VerifyPersonInChargeSignature(rec, pub, sig); err != nil {
		t.Fatalf("competent person rejected at exact 85%%: %v", err)
	}
}

func TestVerifyRejectsTamperedFields(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	rec := PersonInChargeSignRecord{
		ENUEastMeters:    1.0,
		ENUNorthMeters:   2.0,
		ENUUpMeters:      3.0,
		TensorPackDigest: [32]byte{7},
		Role:             RoleAppointedPerson,
	}
	sig := signRec(t, priv, &rec)

	tamper := []func(*PersonInChargeSignRecord){
		func(r *PersonInChargeSignRecord) { r.TensorPackDigest[0] ^= 0xFF },
		func(r *PersonInChargeSignRecord) { r.Role = RoleCompetentPerson },
		func(r *PersonInChargeSignRecord) { r.ENUEastMeters += 1 },
		func(r *PersonInChargeSignRecord) { r.StatutoryStandard = "BOLLOCKS 1200" },
	}
	for i, fn := range tamper {
		victim := rec
		fn(&victim)
		if err := VerifyPersonInChargeSignature(victim, pub, sig); !errors.Is(err, ErrInvalidSignature) {
			t.Fatalf("tamper %d: expected ErrInvalidSignature, got %v", i, err)
		}
	}
}

func TestVerifyRejectsWrongKeyAndMalformedSignature(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	otherPub, _, _ := ed25519.GenerateKey(rand.Reader)
	rec := PersonInChargeSignRecord{
		ENUEastMeters:    1.0,
		ENUNorthMeters:   2.0,
		ENUUpMeters:      3.0,
		TensorPackDigest: [32]byte{7},
		Role:             RoleLiftDirector,
	}
	sig := signRec(t, priv, &rec)
	if err := VerifyPersonInChargeSignature(rec, otherPub, sig); !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("expected wrong-key rejection, got %v", err)
	}
	if err := VerifyPersonInChargeSignature(rec, pub, sig[:10]); !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("expected truncated-signature rejection, got %v", err)
	}
}

func TestVerifyRejectsBadTimestampENUAndDigest(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	rec := PersonInChargeSignRecord{
		ENUEastMeters:    1.0,
		ENUNorthMeters:   2.0,
		ENUUpMeters:      3.0,
		TensorPackDigest: [32]byte{7},
		Role:             RoleAppointedPerson,
	}
	sig := signRec(t, priv, &rec)

	rec.TimestampUUIDv7 = ""
	if err := VerifyPersonInChargeSignature(rec, pub, sig); !errors.Is(err, ErrInvalidTimestamp) {
		t.Fatalf("expected invalid-timestamp, got %v", err)
	}
	rec.TimestampUUIDv7 = mustV7(t)

	rec.ENUEastMeters, rec.ENUNorthMeters, rec.ENUUpMeters = 0, 0, 0
	if err := VerifyPersonInChargeSignature(rec, pub, sig); !errors.Is(err, ErrEmptyENU) {
		t.Fatalf("expected empty-ENU, got %v", err)
	}
	rec.ENUEastMeters = 1
	rec.TensorPackDigest = [32]byte{}
	if err := VerifyPersonInChargeSignature(rec, pub, sig); !errors.Is(err, ErrMissingTensorPackDigest) {
		t.Fatalf("expected missing-digest, got %v", err)
	}
}

func mustV7(t *testing.T) string {
	t.Helper()
	ts, err := id.NewV7()
	if err != nil {
		t.Fatal(err)
	}
	return ts
}

func TestVerifyRejectsFutureTimestamp(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	rec := PersonInChargeSignRecord{
		ENUEastMeters:    1.0,
		ENUNorthMeters:   2.0,
		ENUUpMeters:      3.0,
		TensorPackDigest: [32]byte{7},
		Role:             RoleAppointedPerson,
	}
	sig := signRec(t, priv, &rec)
	future, err := id.NewV7FromTime(time.Now().Add(10 * time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	rec.TimestampUUIDv7 = future
	if err := VerifyPersonInChargeSignature(rec, pub, sig); !errors.Is(err, ErrFutureTimestamp) {
		t.Fatalf("expected future-timestamp rejection, got %v", err)
	}
}

func TestSignRejectsNonFiniteUtilization(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	rec := PersonInChargeSignRecord{
		ENUEastMeters:      1.0,
		ENUNorthMeters:     2.0,
		ENUUpMeters:        3.0,
		TensorPackDigest:   [32]byte{7},
		Role:               RoleLiftDirector,
		UtilizationPercent: math.NaN(),
	}
	if _, err := (&SigningCeremony{PrivateKey: priv}).Sign(&rec); !errors.Is(err, ErrNonFiniteUtilization) {
		t.Fatalf("expected non-finite utilization rejection, got %v", err)
	}
}

func TestSignInputDeterministic(t *testing.T) {
	rec := PersonInChargeSignRecord{
		TimestampUUIDv7:   "018f0000-0000-7000-8000-000000000000",
		ENUEastMeters:     1.5,
		ENUNorthMeters:    -2.5,
		ENUUpMeters:       3.0,
		TensorPackDigest:  [32]byte{1, 2, 3},
		Role:              RoleLiftDirector,
		StatutoryStandard: "ASME P30.1",
	}
	in1, err := rec.SignInput()
	if err != nil {
		t.Fatal(err)
	}
	in2, err := rec.SignInput()
	if err != nil {
		t.Fatal(err)
	}
	if string(in1) != string(in2) {
		t.Fatal("SignInput must be deterministic")
	}
	if fmt.Sprintf("%s", in1) != "018f0000-0000-7000-8000-000000000000||E=1.500000,N=-2.500000,U=3.000000||0102030000000000000000000000000000000000000000000000000000000000||LIFT_DIRECTOR||ASME P30.1" {
		t.Fatalf("unexpected canonical input: %s", in1)
	}
}
