package federation

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"
)

func TestFederation_SovereignExportAndIngest(t *testing.T) {
	pubSA, privSA, _ := ed25519.GenerateKey(rand.Reader)
	pubEU, _, _ := ed25519.GenerateKey(rand.Reader)

	gateSA := NewFederationGatekeeper(CellSaudiCentral01)
	gateEU := NewFederationGatekeeper(CellEUWest01)

	gateSA.RegisterCell(CellEUWest01, pubEU)
	gateEU.RegisterCell(CellSaudiCentral01, pubSA)

	// 1. Sovereign restricted data breach: SA cannot export restricted data to EU
	_, err := gateSA.DispatchCrossCellEnvelope(
		CellEUWest01,
		"tenant-ksa-01",
		ClassSovereignRestricted,
		"did:integin:asset:drill-01",
		1,
		[]byte("confidential ksa data"),
		privSA,
	)
	if err != ErrSovereignBoundaryBreach {
		t.Fatalf("expected ErrSovereignBoundaryBreach, got %v", err)
	}

	// 2. Anonymized telemetric data: Allowed to replicate cross-cell
	msg, err := gateSA.DispatchCrossCellEnvelope(
		CellEUWest01,
		"tenant-ksa-01",
		ClassAnonymizedTelemetric,
		"did:integin:asset:drill-01",
		1,
		[]byte("temperature=85C,pressure=120bar"),
		privSA,
	)
	if err != nil {
		t.Fatalf("unexpected dispatch error: %v", err)
	}

	// 3. Ingest into EU cell succeeds
	payload, err := gateEU.IngestCrossCellEnvelope(msg)
	if err != nil {
		t.Fatalf("unexpected ingest error: %v", err)
	}
	if string(payload) != "temperature=85C,pressure=120bar" {
		t.Fatalf("payload mismatch: %s", string(payload))
	}

	// 4. Replay attack rejection: Same or lower epoch fails closed
	_, err = gateEU.IngestCrossCellEnvelope(msg)
	if err != ErrReplayDetected {
		t.Fatalf("expected ErrReplayDetected, got %v", err)
	}
}

func BenchmarkFederationIngest_HotPath(b *testing.B) {
	pubSA, privSA, _ := ed25519.GenerateKey(rand.Reader)
	gateEU := NewFederationGatekeeper(CellEUWest01)
	gateEU.RegisterCell(CellSaudiCentral01, pubSA)

	gateSA := NewFederationGatekeeper(CellSaudiCentral01)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		msg, _ := gateSA.DispatchCrossCellEnvelope(
			CellEUWest01,
			"tenant-ksa-01",
			ClassGlobalPublicTrust,
			"did:integin:asset:bench",
			int64(i+1),
			[]byte("status=active"),
			privSA,
		)
		_, _ = gateEU.IngestCrossCellEnvelope(msg)
	}
}
