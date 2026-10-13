package standardsync

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func cacheCards(t *testing.T) []StandardMetadataCard {
	t.Helper()
	cards := indexFixtureCards(20)
	for i := range cards {
		cards[i].LifecycleState = LifecycleActive
	}
	return cards
}

func TestLocalCacheRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cache.json")
	cards := cacheCards(t)

	cache, err := BuildLocalCacheFromCards(cards)
	if err != nil {
		t.Fatalf("BuildLocalCacheFromCards: %v", err)
	}
	if err := cache.Store(path, cards, time.Now()); err != nil {
		t.Fatalf("Store: %v", err)
	}

	loaded, err := OpenLocalCache(path)
	if err != nil {
		t.Fatalf("OpenLocalCache: %v", err)
	}
	got := loaded.Cards()
	if len(got) != len(cards) {
		t.Fatalf("card count = %d, want %d", len(got), len(cards))
	}
	byDID := map[string]StandardMetadataCard{}
	for _, card := range cards {
		byDID[card.StandardDID] = card
	}
	for _, card := range got {
		want, ok := byDID[card.StandardDID]
		if !ok {
			t.Fatalf("cache returned unexpected card %s", card.StandardDID)
		}
		if want.RevisionYear != card.RevisionYear || want.Code != card.Code {
			t.Fatalf("card %s changed after round trip", card.StandardDID)
		}
	}
	if loaded.Index().Len() != len(cards) {
		t.Fatalf("rebuilt index size = %d, want %d", loaded.Index().Len(), len(cards))
	}
}

func TestLocalCacheRejectsCorruption(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cache.json")
	cache, _ := BuildLocalCacheFromCards(cacheCards(t))
	if err := cache.Store(path, cacheCards(t), time.Now()); err != nil {
		t.Fatalf("Store: %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	// Flip one byte inside the checksummed payload.
	raw[len(raw)/2] ^= 0xff
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenLocalCache(path); err == nil {
		t.Fatal("corrupt cache loaded without error")
	}
}

func TestLocalCacheRejectsForeignSchema(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cache.json")
	cache, _ := BuildLocalCacheFromCards(cacheCards(t))
	if err := cache.Store(path, cacheCards(t), time.Now()); err != nil {
		t.Fatal(err)
	}
	// Rewrite the schema version so load must refuse.
	envelope := struct {
		SchemaVersion int                    `json:"schema_version"`
		Cards         []StandardMetadataCard `json:"cards"`
	}{SchemaVersion: cacheSchemaVersion + 1, Cards: cacheCards(t)}
	raw, err := json.Marshal(envelope)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenLocalCache(path); err == nil {
		t.Fatal("foreign-schema cache loaded without error")
	}
}
