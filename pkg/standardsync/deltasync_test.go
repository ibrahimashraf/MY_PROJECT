package standardsync

import (
	"errors"
	"testing"
)

func TestDeltaSyncZeroDiffNoop(t *testing.T) {
	engine, err := NewDeltaSyncEngine([]StandardMetadataCard{
		mustCard(t, "did:integin:standard:iso-4309-2010", "ISO", "4309", 2010, LifecycleActive, "2010-02-15", ""),
		mustCard(t, "did:integin:standard:asme-b30-5-2021", "ASME", "B30.5", 2021, LifecycleActive, "2021-05-01", ""),
	}, 3, 0)
	if err != nil {
		t.Fatalf("NewDeltaSyncEngine: %v", err)
	}

	resp, err := engine.ComputeDelta(DeltaSyncRequest{
		CurrentRevisions: map[string]int{
			"did:integin:standard:iso-4309-2010":   2010,
			"did:integin:standard:asme-b30-5-2021": 2021,
		},
		ClientEpoch: 3,
	})
	if err != nil {
		t.Fatalf("ComputeDelta: %v", err)
	}
	if len(resp.UpdatedStandards) != 0 {
		t.Errorf("zero-diff must emit no updates, got %d", len(resp.UpdatedStandards))
	}
	if len(resp.WithdrawnDIDs) != 0 {
		t.Errorf("zero-diff must emit no withdrawals, got %v", resp.WithdrawnDIDs)
	}
	if resp.NewEpoch != 3 || resp.HasMore {
		t.Errorf("unexpected response: epoch=%d hasMore=%v", resp.NewEpoch, resp.HasMore)
	}
}

func TestDeltaSyncIncrementalUpdates(t *testing.T) {
	engine, err := NewDeltaSyncEngine([]StandardMetadataCard{
		mustCard(t, "did:integin:standard:asme-b30-2-2016", "ASME", "B30.2", 2016, LifecycleSuperseded, "2016-01-01", "did:integin:standard:asme-b30-2-2021"),
		mustCard(t, "did:integin:standard:asme-b30-2-2021", "ASME", "B30.2", 2021, LifecycleActive, "2021-05-01", "did:integin:standard:asme-b30-2-2016"),
		mustCard(t, "did:integin:standard:iso-4309-2010", "ISO", "4309", 2010, LifecycleActive, "2010-02-15", ""),
	}, 5, 0)
	if err != nil {
		t.Fatalf("NewDeltaSyncEngine: %v", err)
	}

	// Client holds the old B30.2 revision and has never seen the ISO card.
	resp, err := engine.ComputeDelta(DeltaSyncRequest{
		CurrentRevisions: map[string]int{
			"did:integin:standard:asme-b30-2-2016": 2016,
		},
		ClientEpoch: 3,
	})
	if err != nil {
		t.Fatalf("ComputeDelta: %v", err)
	}
	if len(resp.UpdatedStandards) != 2 {
		t.Fatalf("expected 2 updated cards, got %d", len(resp.UpdatedStandards))
	}
	for _, card := range resp.UpdatedStandards {
		if card.StandardDID == "did:integin:standard:asme-b30-2-2016" {
			t.Errorf("unchanged superseded card must not be re-sent")
		}
	}
	if len(resp.WithdrawnDIDs) != 0 {
		t.Errorf("superseded-but-present cards are not withdrawn: %v", resp.WithdrawnDIDs)
	}
	if resp.NewEpoch != 5 {
		t.Errorf("NewEpoch = %d, want 5", resp.NewEpoch)
	}
}

func TestDeltaSyncWithdrawnCards(t *testing.T) {
	engine, err := NewDeltaSyncEngine([]StandardMetadataCard{
		mustCard(t, "did:integin:standard:iso-4309-2010", "ISO", "4309", 2010, LifecycleWithdrawn, "2010-02-15", ""),
		mustCard(t, "did:integin:standard:asme-b30-5-2021", "ASME", "B30.5", 2021, LifecycleActive, "2021-05-01", ""),
	}, 7, 0)
	if err != nil {
		t.Fatalf("NewDeltaSyncEngine: %v", err)
	}

	resp, err := engine.ComputeDelta(DeltaSyncRequest{
		CurrentRevisions: map[string]int{
			"did:integin:standard:iso-4309-2010":    2010,
			"did:integin:standard:asme-b30-5-2021":  2021,
			"did:integin:standard:dnv-st-n001-2019": 2019, // vanished from master
			"did:integin:standard:asme-b31-3-2018":  2018, // never existed
		},
		ClientEpoch: 3,
	})
	if err != nil {
		t.Fatalf("ComputeDelta: %v", err)
	}
	want := map[string]bool{
		"did:integin:standard:iso-4309-2010":    true,
		"did:integin:standard:dnv-st-n001-2019": true,
		"did:integin:standard:asme-b31-3-2018":  true,
	}
	if len(resp.WithdrawnDIDs) != len(want) {
		t.Fatalf("withdrawn = %v, want %d entries", resp.WithdrawnDIDs, len(want))
	}
	for _, did := range resp.WithdrawnDIDs {
		if !want[did] {
			t.Errorf("unexpected withdrawn DID %q", did)
		}
	}
	if len(resp.UpdatedStandards) != 0 {
		t.Errorf("client already holds the active cards; no updates expected, got %d", len(resp.UpdatedStandards))
	}
}

func TestDeltaSyncEpochAdvancement(t *testing.T) {
	engine, err := NewDeltaSyncEngine([]StandardMetadataCard{
		mustCard(t, "did:integin:standard:asme-b30-5-2021", "ASME", "B30.5", 2021, LifecycleActive, "2021-05-01", ""),
	}, 9, 0)
	if err != nil {
		t.Fatalf("NewDeltaSyncEngine: %v", err)
	}

	// Older client catches up to the authoritative epoch with no content change.
	resp, err := engine.ComputeDelta(DeltaSyncRequest{
		CurrentRevisions: map[string]int{"did:integin:standard:asme-b30-5-2021": 2021},
		ClientEpoch:      2,
	})
	if err != nil {
		t.Fatalf("ComputeDelta: %v", err)
	}
	if resp.NewEpoch != 9 {
		t.Errorf("NewEpoch = %d, want 9", resp.NewEpoch)
	}
	if len(resp.UpdatedStandards) != 0 {
		t.Errorf("epoch catch-up must not re-send unchanged cards")
	}

	if _, err := engine.ComputeDelta(DeltaSyncRequest{
		CurrentRevisions: map[string]int{},
		ClientEpoch:      10,
	}); !errors.Is(err, ErrClientEpochAhead) {
		t.Errorf("client ahead must be rejected, got %v", err)
	}
}

func TestDeltaSyncJurisdictionScoping(t *testing.T) {
	local := mustCard(t, "did:integin:standard:aramco-sae-j-011-2022", "ARAMCO", "SAE-J-011", 2022, LifecycleActive, "2022-03-01", "")
	local.Tags = []string{"SA"}
	global := mustCard(t, "did:integin:standard:iso-4309-2010", "ISO", "4309", 2010, LifecycleActive, "2010-02-15", "")
	global.Tags = []string{"GLOBAL"}

	engine, err := NewDeltaSyncEngine([]StandardMetadataCard{local, global}, 4, 0)
	if err != nil {
		t.Fatalf("NewDeltaSyncEngine: %v", err)
	}

	resp, err := engine.ComputeDelta(DeltaSyncRequest{
		CurrentRevisions: map[string]int{},
		ClientEpoch:      4,
		Jurisdiction:     "SA",
	})
	if err != nil {
		t.Fatalf("ComputeDelta(SA): %v", err)
	}
	if len(resp.UpdatedStandards) != 2 {
		t.Errorf("SA should receive local + global cards, got %d: %v", len(resp.UpdatedStandards), resp.UpdatedStandards)
	}

	resp, err = engine.ComputeDelta(DeltaSyncRequest{
		CurrentRevisions: map[string]int{},
		ClientEpoch:      4,
		Jurisdiction:     "AE",
	})
	if err != nil {
		t.Fatalf("ComputeDelta(AE): %v", err)
	}
	for _, card := range resp.UpdatedStandards {
		if card.StandardDID == local.StandardDID {
			t.Errorf("AE must not receive the SA-restricted card")
		}
	}
}

func TestDeltaSyncPaginationHasMore(t *testing.T) {
	var cards []StandardMetadataCard
	for year := 2000; year <= 2025; year++ {
		cards = append(cards, mustCard(t,
			"did:integin:standard:asme-b30-47-"+itoa(year), "ASME", "B30.47", year, LifecycleActive,
			"2000-01-15", ""))
	}
	engine, err := NewDeltaSyncEngine(cards, 1, 10)
	if err != nil {
		t.Fatalf("NewDeltaSyncEngine: %v", err)
	}
	resp, err := engine.ComputeDelta(DeltaSyncRequest{
		CurrentRevisions: map[string]int{},
		ClientEpoch:      1,
	})
	if err != nil {
		t.Fatalf("ComputeDelta: %v", err)
	}
	if !resp.HasMore {
		t.Error("expected HasMore for a truncated page")
	}
	if len(resp.UpdatedStandards) != 10 {
		t.Errorf("page size must cap updates at 10, got %d", len(resp.UpdatedStandards))
	}
	if resp.NewEpoch != 1 {
		t.Errorf("NewEpoch = %d, want 1", resp.NewEpoch)
	}

	// Bad request surfaces.
	if _, err := engine.ComputeDelta(DeltaSyncRequest{Jurisdiction: "SAU", ClientEpoch: 1}); err == nil {
		t.Error("3-letter jurisdiction must be rejected")
	}
}
