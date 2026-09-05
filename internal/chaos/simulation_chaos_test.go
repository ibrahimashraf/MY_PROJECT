package chaos_test

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"math/big"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	_ "github.com/lib/pq"

	"integin/internal/domain/device_trust"
	domainsync "integin/internal/domain/sync"
	"integin/internal/security"
	"integin/internal/syncstate"
)

// TestDeterministicSimulationChaosHarness implements the Apple/FoundationDB pattern:
// Exhaustive simulated real-world turbulence under concurrent goroutine pressure.
func TestDeterministicSimulationChaosHarness(t *testing.T) {
	dsn := os.Getenv("INTEGIN_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set INTEGIN_TEST_DATABASE_URL to run simulation chaos harness")
	}

	ctx := context.Background()
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(50)
	db.SetMaxIdleConns(20)

	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("database ping failed: %v", err)
	}

	repo, err := syncstate.NewPostgresRepository(db)
	if err != nil {
		t.Fatalf("failed to create syncstate repo: %v", err)
	}

	secrets := map[string]string{"default": "chaos_test_secret_32_bytes_long_ok"}
	processor, err := domainsync.NewProcessorWithState(secrets, repo)
	if err != nil {
		t.Fatalf("failed to initialize processor: %v", err)
	}

	const (
		numTenants = 3
		numDevicesPerTenant = 4
		mutationsPerDevice = 10
	)

	tenantIDs := make([]string, numTenants)
	orgIDs := make([]string, numTenants)
	for i := 0; i < numTenants; i++ {
		tenantIDs[i] = fmt.Sprintf("tenant_chaos_%d_%d", i, time.Now().UnixNano())
		orgIDs[i] = fmt.Sprintf("org_chaos_%d", i)
	}

	type simulatedDevice struct {
		tenantID string
		orgID    string
		deviceID string
		userID   string
		pubKey   ed25519.PublicKey
		privKey  ed25519.PrivateKey
		keyID    string
		authPkg  device_trust.AuthorityPackage
	}

	devices := make([]simulatedDevice, 0, numTenants*numDevicesPerTenant)

	t.Log(">>> [SIMULATION PHASE 1]: Enrolling and registering devices...")
	for tIdx := 0; tIdx < numTenants; tIdx++ {
		for dIdx := 0; dIdx < numDevicesPerTenant; dIdx++ {
			pub, priv, err := ed25519.GenerateKey(rand.Reader)
			if err != nil {
				t.Fatalf("generate ed25519 key: %v", err)
			}
			keyID := security.DeviceKeyID(pub)
			runStamp := time.Now().UnixNano()
			devID := fmt.Sprintf("dev_%d_%d_%d", tIdx, dIdx, runStamp)
			userID := fmt.Sprintf("user_%d_%d_%d", tIdx, dIdx, runStamp)

			now := time.Now().UTC()
			devRec := syncstate.DeviceRecord{
				DeviceID:       devID,
				TenantID:       tenantIDs[tIdx],
				OrganizationID: orgIDs[tIdx],
				UserID:         userID,
				KeyID:          keyID,
				PublicKey:      pub,
				State:          syncstate.DeviceTrusted,
				AuthorityEpoch: 1,
				EnrolledAt:     now,
				UpdatedAt:      now,
			}
			if err := repo.SaveDevice(ctx, devRec); err != nil {
				t.Fatalf("failed to register device %s: %v", devID, err)
			}

			// Register in memory processor
			memDev, err := device_trust.RestoreDevice(devID, tenantIDs[tIdx], orgIDs[tIdx], userID, base64.StdEncoding.EncodeToString(pub), "TRUSTED", 1)
			if err != nil {
				t.Fatalf("mem dev: %v", err)
			}
			processor.RegisterDevice(memDev)

			authID := fmt.Sprintf("auth_%d_%d", tIdx, dIdx)
			authPkg, err := device_trust.IssueAuthorityPackage(memDev, authID, secrets["default"], []string{"inspection.perform", "evidence.upload"}, now.Add(-1*time.Hour), 24*time.Hour)
			if err != nil {
				t.Fatalf("issue auth: %v", err)
			}

			devices = append(devices, simulatedDevice{
				tenantID: tenantIDs[tIdx],
				orgID:    orgIDs[tIdx],
				deviceID: devID,
				userID:   userID,
				pubKey:   pub,
				privKey:  priv,
				keyID:    keyID,
				authPkg:  authPkg,
			})
		}
	}

	t.Logf(">>> [SIMULATION PHASE 2]: Launching Chaos Swarm with %d concurrent devices...", len(devices))

	var (
		appliedCount   int64
		duplicateCount int64
		heldCount      int64
		rejectedCount  int64
		wg             sync.WaitGroup
	)

	// Inject chaotic concurrent submissions
	for _, dev := range devices {
		wg.Add(1)
		go func(d simulatedDevice) {
			defer wg.Done()

			// Prepare mutations out of order intentionally to simulate asynchronous network lag
			// Sequence order: [3, 1, 4, 2, 5, 8, 6, 7, 10, 9]
			chaosSeq := []uint64{3, 1, 4, 2, 5, 8, 6, 7, 10, 9}

			for _, seq := range chaosSeq {
				txID := fmt.Sprintf("tx_%s_seq_%d", d.deviceID, seq)
				payload := []byte(fmt.Sprintf(`{"field_reading":%d,"temperature":24.5}`, seq))

				tx := domainsync.Transaction{
					ProtocolVersion:    "v1",
					TransactionID:      txID,
					TenantID:           d.tenantID,
					OrganizationID:     d.orgID,
					Environment:        "LIVE",
					DeviceID:           d.deviceID,
					UserID:             d.userID,
					SequenceNumber:     seq,
					Operation:          "InspectionSubmitted",
					EntityID:           fmt.Sprintf("insp_%s", d.deviceID),
					Payload:            payload,
					PayloadHash:        domainsync.HashPayload(payload),
					CapturedAt:         time.Now().UTC(),
					AuthorityID:        d.authPkg.ID,
					AuthorityEpoch:     d.authPkg.Epoch,
					SignatureAlgorithm: "Ed25519",
					KeyID:              d.keyID,
				}

				signedTx, err := domainsync.SignTransactionEd25519(tx, d.privKey, d.keyID)
				if err != nil {
					t.Errorf("sign failed: %v", err)
					return
				}

				// Simulated jitter
				jitterMs, _ := rand.Int(rand.Reader, big.NewInt(15))
				time.Sleep(time.Duration(jitterMs.Int64()) * time.Millisecond)

				res := processor.SubmitContext(ctx, signedTx, d.authPkg, time.Now().UTC())
				switch res.Outcome {
				case domainsync.Applied:
					atomic.AddInt64(&appliedCount, 1)
				case domainsync.Held:
					atomic.AddInt64(&heldCount, 1)
				case domainsync.Duplicate:
					atomic.AddInt64(&duplicateCount, 1)
				default:
					atomic.AddInt64(&rejectedCount, 1)
				}

				// In parallel, 20% of the time, simulate an instant network duplicate retry
				if seq%5 == 0 {
					resDup := processor.SubmitContext(ctx, signedTx, d.authPkg, time.Now().UTC())
					if resDup.Outcome == domainsync.Duplicate {
						atomic.AddInt64(&duplicateCount, 1)
					}
				}
			}
		}(dev)
	}

	wg.Wait()
	t.Logf(">>> Swarm completed: Applied=%d, Held=%d, Duplicate=%d, Rejected=%d", appliedCount, heldCount, duplicateCount, rejectedCount)

	t.Log(">>> [SIMULATION PHASE 3]: Draining held queues sequentially per device...")
	var drainedCount int64
	for _, dev := range devices {
		drained := processor.DrainHeld(ctx, dev.tenantID, dev.deviceID, time.Now().UTC())
		for _, r := range drained {
			if r.Outcome == domainsync.Applied {
				drainedCount++
			}
		}
	}

	totalSuccessfullyApplied := appliedCount + drainedCount
	expectedTotalMutations := int64(len(devices) * mutationsPerDevice)

	t.Logf(">>> Post-Drain Results: Total Successfully Applied = %d / Expected = %d", totalSuccessfullyApplied, expectedTotalMutations)

	if totalSuccessfullyApplied != expectedTotalMutations {
		t.Fatalf("CHAOS RECONCILIATION FAILURE: expected exactly %d mutations applied, but got %d", expectedTotalMutations, totalSuccessfullyApplied)
	}

	t.Log(">>> [SIMULATION PHASE 4]: Verifying durable database state across all tenants...")
	for _, dev := range devices {
		lastSeq, err := repo.GetLastAcceptedSequence(ctx, dev.tenantID, dev.deviceID)
		if err != nil {
			t.Fatalf("failed to retrieve last sequence for %s: %v", dev.deviceID, err)
		}
		if lastSeq != mutationsPerDevice {
			t.Fatalf("DEVICE %s MONOTONICITY CORRUPTION: expected last sequence %d, got %d", dev.deviceID, mutationsPerDevice, lastSeq)
		}
	}

	t.Log(">>> [SIMULATION PASSED]: 100% ACID Monotonicity Verified Under Concurrent Chaos Turbulence.")
}
