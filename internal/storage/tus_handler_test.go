package storage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"
	"time"
)

func newTestTUSManager(t *testing.T, staleTTL time.Duration) *TUSManager {
	t.Helper()
	manager, err := NewTUSManager(t.TempDir(), DefaultTUSChunkSize, staleTTL)
	if err != nil {
		t.Fatal(err)
	}
	return manager
}

func testPayload(size int64) (sum []byte, payload []byte, checksum string) {
	payload = make([]byte, size)
	for i := range payload {
		payload[i] = byte(i * 31)
	}
	digest := sha256.Sum256(payload)
	return digest[:], payload, hex.EncodeToString(digest[:])
}

func TestTUSHappyPathUploadsAndVerifies(t *testing.T) {
	manager := newTestTUSManager(t, time.Hour)
	ctx := context.Background()
	_, payload, checksum := testPayload(DefaultTUSChunkSize + 100)
	id, err := manager.Create(ctx, int64(len(payload)), checksum, "image/jpeg")
	if err != nil {
		t.Fatal(err)
	}
	next, err := manager.Append(ctx, id, 0, payload[:DefaultTUSChunkSize])
	if err != nil {
		t.Fatal(err)
	}
	if next != DefaultTUSChunkSize {
		t.Fatalf("unexpected offset after first chunk: %d", next)
	}
	next, err = manager.Append(ctx, id, next, payload[DefaultTUSChunkSize:])
	if err != nil {
		t.Fatal(err)
	}
	if next != int64(len(payload)) {
		t.Fatalf("unexpected offset after final chunk: %d", next)
	}
	object, err := manager.Complete(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if object.Key != id || object.ContentType != "image/jpeg" || string(object.Data) != string(payload) {
		t.Fatalf("unexpected completed object: %#v", object)
	}
	if _, err := manager.Offset(ctx, id); !errors.Is(err, ErrUploadNotFound) {
		t.Fatalf("session should be reclaimed after completion, got %v", err)
	}
}

func TestTUSResumeAfterDrop(t *testing.T) {
	manager := newTestTUSManager(t, time.Hour)
	ctx := context.Background()
	_, payload, checksum := testPayload(DefaultTUSChunkSize * 3)
	id, err := manager.Create(ctx, int64(len(payload)), checksum, "video/mp4")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Append(ctx, id, 0, payload[:DefaultTUSChunkSize]); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Append(ctx, id, DefaultTUSChunkSize, payload[DefaultTUSChunkSize:DefaultTUSChunkSize*2]); err != nil {
		t.Fatal(err)
	}
	session, err := manager.Offset(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if session.Offset != DefaultTUSChunkSize*2 || session.Size != int64(len(payload)) {
		t.Fatalf("unexpected resume state: %#v", session)
	}
	if _, err := manager.Append(ctx, id, DefaultTUSChunkSize*2, payload[DefaultTUSChunkSize*2:]); err != nil {
		t.Fatal(err)
	}
	object, err := manager.Complete(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if string(object.Data) != string(payload) {
		t.Fatal("resumed upload did not reassemble the declared payload")
	}
}

func TestTUSRejectsWrongOffset(t *testing.T) {
	manager := newTestTUSManager(t, time.Hour)
	ctx := context.Background()
	_, payload, checksum := testPayload(DefaultTUSChunkSize * 2)
	id, err := manager.Create(ctx, int64(len(payload)), checksum, "application/octet-stream")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Append(ctx, id, 0, payload[:DefaultTUSChunkSize]); err != nil {
		t.Fatal(err)
	}
	offset, err := manager.Append(ctx, id, 5, payload[5:DefaultTUSChunkSize])
	if !errors.Is(err, ErrUploadOffsetMismatch) {
		t.Fatalf("wrong offset should be rejected with ErrUploadOffsetMismatch, got %v", err)
	}
	if offset != DefaultTUSChunkSize {
		t.Fatalf("rejected chunk must not advance the offset, got %d", offset)
	}
}

func TestTUSRejectsChecksumMismatch(t *testing.T) {
	manager := newTestTUSManager(t, time.Hour)
	ctx := context.Background()
	_, payload, _ := testPayload(DefaultTUSChunkSize)
	wrong := hex.EncodeToString(make([]byte, 32))
	id, err := manager.Create(ctx, int64(len(payload)), wrong, "image/webp")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Append(ctx, id, 0, payload); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Complete(ctx, id); err == nil {
		t.Fatal("complete with mismatched checksum should fail")
	}
}

func TestTUSAbortAndPurgeStale(t *testing.T) {
	manager := newTestTUSManager(t, 25*time.Millisecond)
	ctx := context.Background()
	_, payload, checksum := testPayload(DefaultTUSChunkSize * 2)
	id, err := manager.Create(ctx, int64(len(payload)), checksum, "image/avif")
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.Abort(ctx, id); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Offset(ctx, id); !errors.Is(err, ErrUploadNotFound) {
		t.Fatalf("aborted session should be gone, got %v", err)
	}
	staleID, err := manager.Create(ctx, int64(len(payload)), checksum, "image/avif")
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(50 * time.Millisecond)
	count, err := manager.PurgeStale(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected 1 stale session reclaimed, got %d", count)
	}
	if _, err := manager.Offset(ctx, staleID); !errors.Is(err, ErrUploadNotFound) {
		t.Fatalf("stale session should be reclaimed, got %v", err)
	}
}

func TestTUSRejectsInvalidCreateAndChunks(t *testing.T) {
	manager := newTestTUSManager(t, time.Hour)
	ctx := context.Background()
	_, payload, checksum := testPayload(64)
	if _, err := manager.Create(ctx, 0, checksum, "image/jpeg"); err == nil {
		t.Fatal("zero size should be rejected")
	}
	if _, err := manager.Create(ctx, -1, checksum, "image/jpeg"); err == nil {
		t.Fatal("negative size should be rejected")
	}
	if _, err := manager.Create(ctx, 64, "not-a-checksum", "image/jpeg"); err == nil {
		t.Fatal("malformed checksum should be rejected")
	}
	id, err := manager.Create(ctx, int64(len(payload)), checksum, "image/jpeg")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Append(ctx, id, 0, nil); err == nil {
		t.Fatal("empty chunk should be rejected")
	}
	oversized := make([]byte, DefaultTUSChunkSize+1)
	if _, err := manager.Append(ctx, id, 0, oversized); err == nil {
		t.Fatal("oversized chunk should be rejected")
	}
	if _, err := manager.Append(ctx, id, -3, payload); err == nil {
		t.Fatal("negative offset should be rejected")
	}
	if _, err := manager.Append(ctx, id, 0, make([]byte, 32)); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Append(ctx, id, 32, payload); err == nil {
		t.Fatal("chunk past declared size should be rejected")
	}
}
