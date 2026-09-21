package domain_test

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"errors"
	"math/big"
	"strings"
	"testing"
	"time"

	"integin/pkg/domain" // Path aligned with go.mod module root
)

// Helper: Generates a test P-256 ECDSA keypair.
func generateTestP256(t *testing.T) (*ecdsa.PrivateKey, *ecdsa.PublicKey) {
	t.Helper()
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate P-256 key: %v", err)
	}
	return priv, &priv.PublicKey
}

// Helper: Generates a test Ed25519 keypair.
func generateTestEd25519(t *testing.T) (ed25519.PrivateKey, ed25519.PublicKey) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate Ed25519 key: %v", err)
	}
	return priv, pub
}

// Helper: Encodes an ECDSA signature to 64-byte raw IEEE P1363 (R || S).
func encodeRawP1363(r, s *big.Int) []byte {
	raw := make([]byte, 64)
	rBytes := r.Bytes()
	sBytes := s.Bytes()
	copy(raw[32-len(rBytes):32], rBytes)
	copy(raw[64-len(sBytes):64], sBytes)
	return raw
}

// =========================================================================
// 1. ISSUANCE POLICY BOUNDARY & PROVENANCE SUITE
// =========================================================================

func TestIssueTimeHorizonToken_PolicyBoundaries(t *testing.T) {
	privECDSA, pubECDSA := generateTestP256(t)
	_, wrongPubECDSA := generateTestP256(t)
	privEd, pubEd := generateTestEd25519(t)
	_, wrongPubEd := generateTestEd25519(t)

	now := time.Now().UTC()

	tests := []struct {
		name          string
		criticality   domain.AssetCriticality
		duration      time.Duration
		maxSkew       time.Duration
		expectedError error
	}{
		{
			name:          "LifeSafety_Success",
			criticality:   domain.CriticalityLifeSafety,
			duration:      8 * time.Hour,
			maxSkew:       5 * time.Minute,
			expectedError: nil,
		},
		{
			name:          "LifeSafety_ExceedsMaxHorizon",
			criticality:   domain.CriticalityLifeSafety,
			duration:      8*time.Hour + time.Minute,
			maxSkew:       5 * time.Minute,
			expectedError: domain.ErrHorizonCapExceeded,
		},
		{
			name:          "LifeSafety_ExceedsMaxSkew",
			criticality:   domain.CriticalityLifeSafety,
			duration:      4 * time.Hour,
			maxSkew:       5*time.Minute + time.Second,
			expectedError: domain.ErrSkewCapExceeded,
		},
		{
			name:          "Secondary_Success_MultiDay",
			criticality:   domain.CriticalitySecondary,
			duration:      72 * time.Hour,
			maxSkew:       30 * time.Minute,
			expectedError: nil,
		},
		{
			name:          "Secondary_ExceedsMaxHorizon",
			criticality:   domain.CriticalitySecondary,
			duration:      73 * time.Hour,
			maxSkew:       10 * time.Minute,
			expectedError: domain.ErrHorizonCapExceeded,
		},
		{
			name:          "Unregistered_Criticality_Rejection",
			criticality:   domain.AssetCriticality("TIER_3_UNSUPPORTED"),
			duration:      1 * time.Hour,
			maxSkew:       1 * time.Minute,
			expectedError: domain.ErrUnknownCriticality,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tok, err := domain.IssueTimeHorizonToken(
				"tok-001",
				"wp-001",
				"insp-001",
				tc.criticality,
				now,
				1000000,
				tc.maxSkew,
				tc.duration,
				10,
				privECDSA,
			)

			if tc.expectedError != nil {
				if !errors.Is(err, tc.expectedError) {
					t.Fatalf("expected error %v, got %v", tc.expectedError, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Valid edge epoch
			if err := tok.VerifyToken(pubECDSA, 10); err != nil {
				t.Fatalf("token verification failed: %v", err)
			}
			// Stale epoch refusal
			if err := tok.VerifyToken(pubECDSA, 11); !errors.Is(err, domain.ErrStaleAuthorityEpoch) {
				t.Fatalf("expected ErrStaleAuthorityEpoch, got %v", err)
			}
			// Forgery detection: ECDSA wrong key
			if err := tok.VerifyToken(wrongPubECDSA, 10); !errors.Is(err, domain.ErrInvalidIssuerSignature) {
				t.Fatalf("expected ErrInvalidIssuerSignature on forged ECDSA token, got %v", err)
			}
		})
	}

	t.Run("Ed25519_Issuer_Mint_And_Verify_Provenance", func(t *testing.T) {
		tokEd, err := domain.IssueTimeHorizonToken(
			"tok-ed-01",
			"wp-001",
			"insp-001",
			domain.CriticalityLifeSafety,
			now,
			1000000,
			5*time.Minute,
			4*time.Hour,
			5,
			privEd,
		)
		if err != nil {
			t.Fatalf("failed to mint Ed25519 token: %v", err)
		}

		if err := tokEd.VerifyToken(pubEd, 5); err != nil {
			t.Fatalf("Ed25519 token verification failed: %v", err)
		}
		// Forgery detection: Ed25519 wrong key
		if err := tokEd.VerifyToken(wrongPubEd, 5); !errors.Is(err, domain.ErrInvalidIssuerSignature) {
			t.Fatalf("expected ErrInvalidIssuerSignature on forged Ed25519 token, got %v", err)
		}
	})

	t.Run("Malformed_Issuer_Signature_Matches_Both_Sentinels", func(t *testing.T) {
		tok := domain.TimeHorizonToken{
			AuthorityEpoch:  5,
			IssuerSignature: []byte{0xDE, 0xAD, 0xBE, 0xEF},
		}
		_, pubECDSA2 := generateTestP256(t)

		err := tok.VerifyToken(pubECDSA2, 5)
		if !errors.Is(err, domain.ErrInvalidIssuerSignature) || !errors.Is(err, domain.ErrMalformedP256Sig) {
			t.Fatalf("expected both ErrInvalidIssuerSignature and ErrMalformedP256Sig, got: %v", err)
		}
	})
}

// =========================================================================
// 2. TOMBSTONE SUPREMACY & FRESHNESS
// =========================================================================

func TestLeaseTombstone_SupremacyAndStaleSequence(t *testing.T) {
	tombstone := domain.LeaseTombstoneRecord{
		SubAssemblyID:       "CRANE-01:BOOM_LACING",
		LeaseEpoch:          4,
		TombstoneSequence:   2,
		RevokedDeviceID:     "device-field-01",
		RevocationTimestamp: time.Now().UTC().Add(-2 * time.Hour),
		AuthorizingDID:      "did:integin:authority:qa-lead",
		Reason:              string(domain.RootCauseShiftHandoverAbandoned),
	}

	t.Run("Suppresses_RevokedDevice_Writes_Under_Same_Or_Past_Epoch", func(t *testing.T) {
		if !tombstone.AssertsSupremacy("device-field-01", 3) {
			t.Fatal("expected supremacy over incoming epoch 3")
		}
		if !tombstone.AssertsSupremacy("device-field-01", 4) {
			t.Fatal("expected supremacy over incoming epoch 4")
		}
	})

	t.Run("Permits_Writes_Under_Newly_Provisioned_Epoch", func(t *testing.T) {
		if tombstone.AssertsSupremacy("device-field-01", 5) {
			t.Fatal("tombstone must not suppress writes under incremented epoch 5")
		}
	})

	t.Run("Unaffected_Unrelated_Device", func(t *testing.T) {
		if tombstone.AssertsSupremacy("device-field-02", 4) {
			t.Fatal("tombstone must not suppress writes from unrevoked devices")
		}
	})
}

// =========================================================================
// 3. TIMEGUARD & CHECKPOINT LEDGER RIG
// =========================================================================

func TestValidateExecutionTime_ColdBootAndDrift(t *testing.T) {
	anchorWall := time.Now().UTC().Add(-1 * time.Hour)
	anchorMono := int64(100_000_000_000) // 100s baseline

	tok := domain.TimeHorizonToken{
		CriticalityClass: domain.CriticalityLifeSafety, // MaxHorizon: 8h, MaxRebootGrace: 15m
		AnchorWallTime:   anchorWall,
		AnchorMonoNanos:  anchorMono,
		MaxSkewAllowed:   5 * time.Minute,
		HorizonDuration:  4 * time.Hour,
	}

	t.Run("Normal_Monotonic_Execution_Passes", func(t *testing.T) {
		currentWall := anchorWall.Add(1 * time.Hour)
		currentMono := anchorMono + int64(1*time.Hour)

		checkpoint := domain.CheckpointLedgerEntry{
			WallTimestamp:  currentWall.Add(-1 * time.Minute),
			MonotonicNanos: currentMono - int64(1*time.Minute),
			BootSessionID:  "boot-session-a",
		}
		rebootState := domain.RebootRecoveryState{
			ActiveBootSessionID: "boot-session-a",
		}

		err := tok.ValidateExecutionTime(currentWall, currentMono, checkpoint, rebootState)
		if err != nil {
			t.Fatalf("expected valid execution, got: %v", err)
		}
	})

	t.Run("Wall_Clock_Negative_Drift_Rollback_Fails", func(t *testing.T) {
		currentWall := anchorWall.Add(-10 * time.Second)
		currentMono := anchorMono + int64(10*time.Minute)

		checkpoint := domain.CheckpointLedgerEntry{
			WallTimestamp:  anchorWall,
			MonotonicNanos: anchorMono,
			BootSessionID:  "boot-session-a",
		}
		rebootState := domain.RebootRecoveryState{
			ActiveBootSessionID: "boot-session-a",
		}

		err := tok.ValidateExecutionTime(currentWall, currentMono, checkpoint, rebootState)
		if !errors.Is(err, domain.ErrWallClockSkewExceeded) {
			t.Fatalf("expected ErrWallClockSkewExceeded on negative wall drift, got: %v", err)
		}
	})

	t.Run("Monotonic_Sequence_Broken_Without_Reboot", func(t *testing.T) {
		currentWall := anchorWall.Add(10 * time.Minute)
		currentMono := anchorMono - 1000

		checkpoint := domain.CheckpointLedgerEntry{
			WallTimestamp:  anchorWall.Add(9 * time.Minute),
			MonotonicNanos: anchorMono - 2000,
			BootSessionID:  "boot-session-a",
		}
		rebootState := domain.RebootRecoveryState{
			ActiveBootSessionID: "boot-session-a",
		}

		err := tok.ValidateExecutionTime(currentWall, currentMono, checkpoint, rebootState)
		if !errors.Is(err, domain.ErrMonotonicBroken) {
			t.Fatalf("expected ErrMonotonicBroken, got: %v", err)
		}
	})

	t.Run("Monotonic_Horizon_Expired_Fails", func(t *testing.T) {
		currentWall := anchorWall.Add(4 * time.Hour)
		currentMono := anchorMono + int64(4*time.Hour) + int64(time.Minute)

		checkpoint := domain.CheckpointLedgerEntry{
			WallTimestamp:  anchorWall.Add(4 * time.Hour),
			MonotonicNanos: currentMono - int64(time.Minute),
			BootSessionID:  "boot-session-a",
		}
		rebootState := domain.RebootRecoveryState{
			ActiveBootSessionID: "boot-session-a",
		}

		err := tok.ValidateExecutionTime(currentWall, currentMono, checkpoint, rebootState)
		if !errors.Is(err, domain.ErrHorizonExpired) {
			t.Fatalf("expected ErrHorizonExpired, got: %v", err)
		}
	})

	t.Run("Cold_Boot_Without_Valid_Session_Detects_Tamper", func(t *testing.T) {
		currentWall := anchorWall.Add(30 * time.Minute)
		currentMono := int64(5_000_000_000)

		checkpoint := domain.CheckpointLedgerEntry{
			WallTimestamp:  anchorWall.Add(25 * time.Minute),
			MonotonicNanos: anchorMono + int64(25*time.Minute),
			BootSessionID:  "boot-session-a",
		}
		rebootState := domain.RebootRecoveryState{
			ActiveBootSessionID: "",
		}

		err := tok.ValidateExecutionTime(currentWall, currentMono, checkpoint, rebootState)
		if !errors.Is(err, domain.ErrBootTamperDetected) {
			t.Fatalf("expected ErrBootTamperDetected, got: %v", err)
		}
	})

	t.Run("Cold_Boot_Within_Reboot_Grace_Passes", func(t *testing.T) {
		rebootWall := anchorWall.Add(20 * time.Minute)
		evalWall := rebootWall.Add(10 * time.Minute) // 10 min elapsed (grace is 15 min)
		currentMono := int64(600_000_000_000)        // 10 min monotonic counter

		checkpoint := domain.CheckpointLedgerEntry{
			WallTimestamp:  rebootWall,
			MonotonicNanos: anchorMono + int64(20*time.Minute),
			BootSessionID:  "boot-session-a",
		}
		rebootState := domain.RebootRecoveryState{
			ActiveBootSessionID: "boot-session-b",
		}

		err := tok.ValidateExecutionTime(evalWall, currentMono, checkpoint, rebootState)
		if err != nil {
			t.Fatalf("expected reboot recovery to pass within grace window, got: %v", err)
		}
	})

	t.Run("Cold_Boot_Exceeding_Reboot_Grace_Fails", func(t *testing.T) {
		rebootWall := anchorWall.Add(30 * time.Minute)
		currentWall := rebootWall.Add(16 * time.Minute) // 16 min elapsed (grace is 15 min)
		currentMono := int64(960_000_000_000)

		checkpoint := domain.CheckpointLedgerEntry{
			WallTimestamp:  rebootWall,
			MonotonicNanos: anchorMono + int64(30*time.Minute),
			BootSessionID:  "boot-session-a",
		}
		rebootState := domain.RebootRecoveryState{
			ActiveBootSessionID: "boot-session-b",
		}

		err := tok.ValidateExecutionTime(currentWall, currentMono, checkpoint, rebootState)
		if !errors.Is(err, domain.ErrRebootGraceExpired) {
			t.Fatalf("expected ErrRebootGraceExpired, got: %v", err)
		}
	})

	t.Run("Remaining_Horizon_Clamps_Grace_Window", func(t *testing.T) {
		rebootWall := anchorWall.Add(4*time.Hour - 5*time.Minute)
		currentWall := rebootWall.Add(6 * time.Minute) // 6m > 5m remaining horizon
		currentMono := int64(360_000_000_000)

		checkpoint := domain.CheckpointLedgerEntry{
			WallTimestamp:  rebootWall,
			MonotonicNanos: anchorMono + int64(4*time.Hour-5*time.Minute),
			BootSessionID:  "boot-session-a",
		}
		rebootState := domain.RebootRecoveryState{
			ActiveBootSessionID: "boot-session-b",
		}

		err := tok.ValidateExecutionTime(currentWall, currentMono, checkpoint, rebootState)
		if !errors.Is(err, domain.ErrRebootGraceExpired) {
			t.Fatalf("expected ErrRebootGraceExpired due to horizon clamp, got: %v", err)
		}
	})

	t.Run("Zero_Value_Checkpoint_Fails_Closed", func(t *testing.T) {
		// Legacy device without checkpoint support: zero checkpoint entry.
		// Fails closed with the dedicated sentinel (not inside duration
		// math, where year-1 timestamps overflow int64 horizons).
		zeroCheckpoint := domain.CheckpointLedgerEntry{}
		rebootState := domain.RebootRecoveryState{
			ActiveBootSessionID: "boot-session-legacy",
		}

		currentWall := anchorWall.Add(15 * time.Minute)
		currentMono := anchorMono + int64(15*time.Minute)

		err := tok.ValidateExecutionTime(currentWall, currentMono, zeroCheckpoint, rebootState)
		if !errors.Is(err, domain.ErrCheckpointMissing) {
			t.Fatalf("expected legacy checkpoint-less device to fail closed, got: %v", err)
		}
	})
}

// =========================================================================
// 4. HARDWARE SEAL NORMALIZATION & INVERTED STATUTORY GATE
// =========================================================================

func TestSubmissionSeal_Verify_P256NormalizationAndGate(t *testing.T) {
	appPriv, appPub := generateTestEd25519(t)
	hwPriv, hwPub := generateTestP256(t)

	deviceDID := "did:integin:device:rugged-tablet-01"
	tokenID := "tok-test-44"
	leaseEpoch := uint64(5)
	payload := []byte(`{"action":"LOAD_TEST","asset":"PADEYE-04","load_tons":166.4}`)

	appSig := ed25519.Sign(appPriv, payload)

	seal := domain.SubmissionSeal{
		DeviceKeyDID:  deviceDID,
		TokenID:       tokenID,
		LeaseEpoch:    leaseEpoch,
		DataPayload:   payload,
		AppEd25519Sig: appSig,
	}
	digest := seal.DeriveCompositeDigest()

	// 1. Generate ASN.1 DER signature (Apple Secure Enclave)
	derSig, err := ecdsa.SignASN1(rand.Reader, hwPriv, digest)
	if err != nil {
		t.Fatalf("failed to sign ASN1: %v", err)
	}

	// 2. Generate raw IEEE P1363 64-byte signature (Android StrongBox)
	rRaw, sRaw, err := ecdsa.Sign(rand.Reader, hwPriv, digest)
	if err != nil {
		t.Fatalf("failed signing digest: %v", err)
	}
	rawSig := encodeRawP1363(rRaw, sRaw)

	statutoryRoute := "/api/v1/field/inspections/submit"
	allowlistedRoute := "/api/v1/field/telemetry"

	t.Run("Normalizes_Apple_SecureEnclave_ASN1_DER", func(t *testing.T) {
		sealDER := seal
		sealDER.HwP256Sig = derSig

		err := sealDER.Verify(statutoryRoute, deviceDID, tokenID, leaseEpoch, appPub, hwPub)
		if err != nil {
			t.Fatalf("ASN.1 DER verification failed: %v", err)
		}
	})

	t.Run("Normalizes_Android_StrongBox_Raw_IEEE_P1363", func(t *testing.T) {
		sealRaw := seal
		sealRaw.HwP256Sig = rawSig

		err = sealRaw.Verify(statutoryRoute, deviceDID, tokenID, leaseEpoch, appPub, hwPub)
		if err != nil {
			t.Fatalf("Raw IEEE P1363 verification failed: %v", err)
		}
	})

	t.Run("Payload_Tamper_Fails_Ed25519", func(t *testing.T) {
		tamperedSeal := seal
		tamperedSeal.HwP256Sig = derSig
		tamperedSeal.DataPayload = []byte(`{"action":"LOAD_TEST","asset":"PADEYE-04","load_tons":999.9}`)

		err := tamperedSeal.Verify(statutoryRoute, deviceDID, tokenID, leaseEpoch, appPub, hwPub)
		if !errors.Is(err, domain.ErrInvalidSeal) {
			t.Fatalf("expected ErrInvalidSeal on tampered payload, got: %v", err)
		}
	})

	t.Run("Statutory_Route_Rejects_Missing_Hardware_Seal", func(t *testing.T) {
		sealNoHW := seal
		sealNoHW.HwP256Sig = nil

		err := sealNoHW.Verify(statutoryRoute, deviceDID, tokenID, leaseEpoch, appPub, hwPub)
		if !errors.Is(err, domain.ErrStatutoryHardwareSealMissing) {
			t.Fatalf("expected ErrStatutoryHardwareSealMissing, got: %v", err)
		}
	})

	t.Run("Statutory_Route_Rejects_Nil_Hardware_Key", func(t *testing.T) {
		sealWithSig := seal
		sealWithSig.HwP256Sig = derSig

		err := sealWithSig.Verify(statutoryRoute, deviceDID, tokenID, leaseEpoch, appPub, nil)
		if !errors.Is(err, domain.ErrStatutoryHardwareSealMissing) {
			t.Fatalf("expected ErrStatutoryHardwareSealMissing on nil HW key, got: %v", err)
		}
	})

	t.Run("Rejects_Malformed_P256_Signature_Bytes", func(t *testing.T) {
		sealMalformed := seal
		sealMalformed.HwP256Sig = []byte{0x01, 0x02, 0x03, 0x04}

		err := sealMalformed.Verify(statutoryRoute, deviceDID, tokenID, leaseEpoch, appPub, hwPub)
		if !errors.Is(err, domain.ErrInvalidSeal) {
			t.Fatalf("expected ErrInvalidSeal on malformed signature bytes, got: %v", err)
		}
		if !errors.Is(err, domain.ErrMalformedP256Sig) {
			t.Fatalf("expected ErrMalformedP256Sig in error chain, got: %v", err)
		}
	})

	t.Run("Allowlisted_Route_Permits_Ed25519_Alone", func(t *testing.T) {
		sealAllowlisted := seal
		sealAllowlisted.HwP256Sig = nil

		err := sealAllowlisted.Verify(allowlistedRoute, deviceDID, tokenID, leaseEpoch, appPub, nil)
		if err != nil {
			t.Fatalf("allowlisted route must not require HW seal, got: %v", err)
		}
	})

	t.Run("Rejects_Domain_Context_Mismatch_Replay", func(t *testing.T) {
		sealDER := seal
		sealDER.HwP256Sig = derSig

		// Wrong TokenID
		err := sealDER.Verify(statutoryRoute, deviceDID, "wrong-token-id", leaseEpoch, appPub, hwPub)
		if !errors.Is(err, domain.ErrDomainContextMismatch) {
			t.Fatalf("expected ErrDomainContextMismatch on token mismatch, got: %v", err)
		}

		// Wrong LeaseEpoch
		err = sealDER.Verify(statutoryRoute, deviceDID, tokenID, leaseEpoch+1, appPub, hwPub)
		if !errors.Is(err, domain.ErrDomainContextMismatch) {
			t.Fatalf("expected ErrDomainContextMismatch on lease epoch mismatch, got: %v", err)
		}

		// Wrong Device DID
		err = sealDER.Verify(statutoryRoute, "did:integin:device:attacker", tokenID, leaseEpoch, appPub, hwPub)
		if !errors.Is(err, domain.ErrDomainContextMismatch) {
			t.Fatalf("expected ErrDomainContextMismatch on DID mismatch, got: %v", err)
		}
	})
}

// =========================================================================
// 5. GRANULAR COMPETENCY EXECUTION GATE
// =========================================================================

func TestEvaluateExecutionGate_GranularCompetency(t *testing.T) {
	now := time.Now().UTC()
	inspectorID := "insp-leims-77"

	lease := domain.SubAssemblyLease{
		SubAssemblyID:    "CRANE-01:HOIST_ROPE",
		InspectorID:      inspectorID,
		RequiredSkill:    "LEEA-MR",
		CriticalityClass: domain.CriticalityLifeSafety,
	}

	token := domain.TimeHorizonToken{
		InspectorID:      inspectorID,
		CriticalityClass: domain.CriticalityLifeSafety,
	}

	competencies := []domain.InspectorCompetency{
		{
			SkillCode:    "LEEA-MR",
			ValidFrom:    now.Add(-30 * 24 * time.Hour),
			ValidThrough: now.Add(30 * 24 * time.Hour),
			IsRevoked:    false,
		},
		{
			SkillCode:    "NDT-UT-L2",
			ValidFrom:    now.Add(-60 * 24 * time.Hour),
			ValidThrough: now.Add(-2 * time.Hour),
			IsRevoked:    false,
		},
		{
			SkillCode:    "NDT-MT-L2",
			ValidFrom:    now.Add(-30 * 24 * time.Hour),
			ValidThrough: now.Add(30 * 24 * time.Hour),
			IsRevoked:    true,
		},
		{
			SkillCode:    "API-SI-L1",
			ValidFrom:    now.Add(24 * time.Hour),
			ValidThrough: now.Add(365 * 24 * time.Hour),
			IsRevoked:    false,
		},
	}

	t.Run("Valid_Competency_Passes", func(t *testing.T) {
		res := domain.EvaluateExecutionGate(inspectorID, lease, token, competencies, now)
		if !res.Allowed || res.PartialBlock {
			t.Fatalf("expected allowed execution, got: %+v", res)
		}
	})

	t.Run("Unknown_Criticality_Blocks_Loudly", func(t *testing.T) {
		bogusLease := lease
		bogusLease.CriticalityClass = domain.AssetCriticality("BOGUS_CLASS")

		bogusToken := token
		bogusToken.CriticalityClass = domain.AssetCriticality("BOGUS_CLASS")

		res := domain.EvaluateExecutionGate(inspectorID, bogusLease, bogusToken, competencies, now)
		if res.Allowed || !res.PartialBlock {
			t.Fatalf("expected loud partial block on unknown criticality class, got: %+v", res)
		}
		if !strings.Contains(res.FailureReason, "invalid or unregistered criticality class") {
			t.Fatalf("expected unregistered criticality failure reason, got: %q", res.FailureReason)
		}
	})

	t.Run("Revoked_Competency_Blocks", func(t *testing.T) {
		revokedLease := lease
		revokedLease.RequiredSkill = "NDT-MT-L2"

		res := domain.EvaluateExecutionGate(inspectorID, revokedLease, token, competencies, now)
		if res.Allowed || !res.PartialBlock {
			t.Fatalf("expected partial block on revoked skill, got: %+v", res)
		}
	})

	t.Run("Missing_Competency_Record_Blocks", func(t *testing.T) {
		missingLease := lease
		missingLease.RequiredSkill = "UNKNOWN-SKILL-99"

		res := domain.EvaluateExecutionGate(inspectorID, missingLease, token, competencies, now)
		if res.Allowed || !res.PartialBlock {
			t.Fatalf("expected partial block on missing skill record, got: %+v", res)
		}
	})

	t.Run("Not_Yet_Valid_Competency_Blocks", func(t *testing.T) {
		futureLease := lease
		futureLease.RequiredSkill = "API-SI-L1"

		res := domain.EvaluateExecutionGate(inspectorID, futureLease, token, competencies, now)
		if res.Allowed || !res.PartialBlock {
			t.Fatalf("expected partial block on future competency, got: %+v", res)
		}
	})

	t.Run("Inspector_ID_Mismatch_Blocks", func(t *testing.T) {
		res := domain.EvaluateExecutionGate("imposter-inspector", lease, token, competencies, now)
		if res.Allowed || !res.PartialBlock {
			t.Fatalf("expected block on inspector mismatch, got: %+v", res)
		}
	})

	t.Run("Expired_Competency_Zero_Slack_LifeSafety_Blocks", func(t *testing.T) {
		utLease := lease
		utLease.RequiredSkill = "NDT-UT-L2"

		res := domain.EvaluateExecutionGate(inspectorID, utLease, token, competencies, now)
		if res.Allowed || !res.PartialBlock {
			t.Fatalf("expected block on expired NDT skill, got: %+v", res)
		}
	})

	t.Run("Expired_Competency_Secondary_With_24h_Slack_Passes", func(t *testing.T) {
		secondaryLease := lease
		secondaryLease.RequiredSkill = "NDT-UT-L2"
		secondaryLease.CriticalityClass = domain.CriticalitySecondary

		secondaryToken := token
		secondaryToken.CriticalityClass = domain.CriticalitySecondary

		res := domain.EvaluateExecutionGate(inspectorID, secondaryLease, secondaryToken, competencies, now)
		if !res.Allowed || res.PartialBlock {
			t.Fatalf("expected slack allowance on secondary asset, got: %+v", res)
		}
	})
}

// =========================================================================
// 6. IN-CHAIN ESCROW OVERRIDE QUORUM
// =========================================================================

func TestEscrowOverrideRecord_ValidateQuorum(t *testing.T) {
	tdPriv, tdPub := generateTestP256(t)
	ssoPriv, ssoPub := generateTestP256(t)

	tombstoneTime := time.Now().UTC().Add(-5 * time.Hour)
	edgeArrivalTime := time.Now().UTC()
	configuredSLA := 4 * time.Hour
	storedSeq := uint64(3)

	rec := domain.EscrowOverrideRecord{
		OverrideID:          "ovr-999",
		SubAssemblyID:       "CRANE-01:SLEW_RING",
		LeaseEpoch:          2,
		TombstoneSequence:   storedSeq,
		DraftWorkOrderID:    "draft-wo-41",
		OriginalInspectorID: "insp-evacuated",
		TechnicalDirectorID: "user-td-01",
		TdKeyDID:            "did:integin:key:td-01",
		SiteSafetyOfficerID: "user-sso-01",
		SsoKeyDID:           "did:integin:key:sso-01",
		RootCause:           domain.RootCauseMedicalEvacuation,
		PermanentFlag:       true,
	}

	digest := rec.CanonicalDigest()
	tdSig, err := ecdsa.SignASN1(rand.Reader, tdPriv, digest)
	if err != nil {
		t.Fatalf("failed to sign TD signature: %v", err)
	}
	ssoSig, err := ecdsa.SignASN1(rand.Reader, ssoPriv, digest)
	if err != nil {
		t.Fatalf("failed to sign SSO signature: %v", err)
	}

	rec.TdSignature = tdSig
	rec.SsoSignature = ssoSig

	t.Run("Valid_Quorum_SLA_Met_Passes", func(t *testing.T) {
		err := rec.ValidateQuorum(tdPub, ssoPub, configuredSLA, storedSeq, tombstoneTime, edgeArrivalTime)
		if err != nil {
			t.Fatalf("expected valid quorum, got: %v", err)
		}
	})

	t.Run("Fails_If_SLA_Not_Elapsed", func(t *testing.T) {
		prematureArrival := tombstoneTime.Add(2 * time.Hour)
		err := rec.ValidateQuorum(tdPub, ssoPub, configuredSLA, storedSeq, tombstoneTime, prematureArrival)
		if err == nil {
			t.Fatal("expected SLA threshold error, got nil")
		}
	})

	t.Run("Fails_On_Stale_Tombstone_Sequence_Replay", func(t *testing.T) {
		err := rec.ValidateQuorum(tdPub, ssoPub, configuredSLA, storedSeq+1, tombstoneTime, edgeArrivalTime)
		if !errors.Is(err, domain.ErrTombstoneStaleSequence) {
			t.Fatalf("expected ErrTombstoneStaleSequence, got: %v", err)
		}
	})

	t.Run("Fails_If_PermanentFlag_Unset", func(t *testing.T) {
		recUnset := rec
		recUnset.PermanentFlag = false

		err := recUnset.ValidateQuorum(tdPub, ssoPub, configuredSLA, storedSeq, tombstoneTime, edgeArrivalTime)
		if !errors.Is(err, domain.ErrEscrowPermanentFlagRequired) {
			t.Fatalf("expected ErrEscrowPermanentFlagRequired, got: %v", err)
		}
	})

	t.Run("Fails_On_Empty_Signer_DIDs", func(t *testing.T) {
		recNoDID := rec
		recNoDID.TdKeyDID = ""

		err := recNoDID.ValidateQuorum(tdPub, ssoPub, configuredSLA, storedSeq, tombstoneTime, edgeArrivalTime)
		if err == nil {
			t.Fatal("expected error on empty TD DID, got nil")
		}
	})

	t.Run("Fails_On_Tampered_Signatures", func(t *testing.T) {
		recTampered := rec
		tamperedSig := make([]byte, len(tdSig))
		copy(tamperedSig, tdSig)
		tamperedSig[len(tamperedSig)-1] ^= 0xFF
		recTampered.TdSignature = tamperedSig

		err := recTampered.ValidateQuorum(tdPub, ssoPub, configuredSLA, storedSeq, tombstoneTime, edgeArrivalTime)
		if err == nil {
			t.Fatal("expected cryptographic failure on tampered signature, got nil")
		}
	})
}
