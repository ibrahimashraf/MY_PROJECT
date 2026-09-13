package storage

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeOrphanSession(t *testing.T, dir, id string, chunk []byte, size int64, checksum string, modTime time.Time) {
	t.Helper()
	sidecar := tusSidecar{ID: id, Size: size, Checksum: checksum, Offset: 0, ContentType: "application/octet-stream"}
	data, err := json.Marshal(sidecar)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, id+".json"), data, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, id), chunk, 0o600); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{id, id + ".json"} {
		if err := os.Chtimes(filepath.Join(dir, name), modTime, modTime); err != nil {
			t.Fatal(err)
		}
	}
}

func TestTUSSweepPrunesStaleZeroByteOrphan(t *testing.T) {
	dir := t.TempDir()
	manager, err := NewTUSManager(dir, 0, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	_, _, checksum := testPayload(100)
	old := time.Now().UTC().Add(-48 * time.Hour)
	// A crash-left zero-byte chunk with a sidecar describing 100 bytes: stale,
	// never committed a byte, and untouched past the TTL — must be pruned, not resumed.
	writeOrphanSession(t, dir, "stale-orphan", nil, 100, checksum, old)

	report, err := manager.Sweep(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if report.Resumed != 0 || report.Deleted != 1 {
		t.Fatalf("stale orphan must be pruned, got %+v", report)
	}
	for _, name := range []string{"stale-orphan", "stale-orphan.json"} {
		if _, err := os.Stat(filepath.Join(dir, name)); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("stale orphan file %s must be deleted, got %v", name, err)
		}
	}
}

func TestTUSSweepResumesFreshIncompleteOrphan(t *testing.T) {
	dir := t.TempDir()
	manager, err := NewTUSManager(dir, 0, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	_, payload, checksum := testPayload(100)
	// Fresh crash orphan: 50 of 100 bytes received, untouched for minutes only.
	writeOrphanSession(t, dir, "fresh-orphan", payload[:50], 100, checksum, time.Now().UTC().Add(-5*time.Minute))

	report, err := manager.Sweep(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if report.Resumed != 1 || report.Deleted != 0 {
		t.Fatalf("fresh orphan must be resumed, got %+v", report)
	}
	session, err := manager.Offset(context.Background(), "fresh-orphan")
	if err != nil {
		t.Fatal(err)
	}
	if session.Offset != 50 || session.Size != 100 {
		t.Fatalf("resume offset/size wrong: %+v", session)
	}
}

func TestTUSSweepPurgesStaleLiveSession(t *testing.T) {
	manager := newTestTUSManager(t, time.Hour)
	ctx := context.Background()
	_, payload, checksum := testPayload(DefaultTUSChunkSize)
	id, err := manager.Create(ctx, int64(len(payload)), checksum, "video/mp4")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Append(ctx, id, 0, payload[:100]); err != nil {
		t.Fatal(err)
	}
	manager.mu.Lock()
	manager.sessions[id].UploadSession.UpdatedAt = time.Now().UTC().Add(-2 * time.Hour)
	manager.mu.Unlock()

	report, err := manager.Sweep(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if report.Purged != 1 || report.Active != 0 {
		t.Fatalf("stale live session must be purged, got %+v", report)
	}
	if _, err := manager.Offset(ctx, id); !errors.Is(err, ErrUploadNotFound) {
		t.Fatalf("purged session must be gone, got %v", err)
	}
}

func TestTUSSweepReportsActiveUploadCount(t *testing.T) {
	manager := newTestTUSManager(t, time.Hour)
	ctx := context.Background()
	_, _, checksum := testPayload(10)
	if _, err := manager.Create(ctx, 10, checksum, "image/png"); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Create(ctx, 10, checksum, "image/png"); err != nil {
		t.Fatal(err)
	}
	report, err := manager.Sweep(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if report.Active != 2 || report.Resumed != 0 || report.Purged != 0 {
		t.Fatalf("active count wrong, got %+v", report)
	}
}

func TestTUSJanitorRunsSweepAndStopsOnCancel(t *testing.T) {
	dir := t.TempDir()
	manager, err := NewTUSManager(dir, 0, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	_, _, checksum := testPayload(100)
	writeOrphanSession(t, dir, "janitor-orphan", nil, 100, checksum, time.Now().UTC().Add(-72*time.Hour))

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- manager.RunJanitor(ctx, time.Millisecond, nil) }()

	deadline := time.After(3 * time.Second)
	for {
		if _, err := os.Stat(filepath.Join(dir, "janitor-orphan")); errors.Is(err, os.ErrNotExist) {
			break
		}
		select {
		case <-deadline:
			cancel()
			t.Fatal("janitor never pruned the stale orphan")
		case <-time.After(10 * time.Millisecond):
		}
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("janitor must stop with context.Canceled, got %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("janitor did not stop after cancellation")
	}
}
