package storage

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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

func TestTUSRecoverOrphansDeletesStaleChunkFiles(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"deadbeefdeadbeefdeadbeefdeadbeef", "cafebabecafebabecafebabecafebabe.part"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("partial chunk bytes"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	manager, err := NewTUSManager(dir, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	_, payload, checksum := testPayload(64)
	id, err := manager.Create(ctx, int64(len(payload)), checksum, "image/jpeg")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Append(ctx, id, 0, payload); err != nil {
		t.Fatal(err)
	}
	// Simulate a second start: pull the manager back to construction state.
	recovered, err := NewTUSManager(dir, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := recovered.Offset(ctx, id); !errors.Is(err, ErrUploadNotFound) {
		t.Fatalf("orphaned session should not be re-attached without a sidecar, got %v", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("all orphaned chunk files should be deleted on restart, found %d", len(entries))
	}
}

func TestTUSRecoverOrphansKeepsLiveSessions(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "orphan-file"), []byte("leftover"), 0o600); err != nil {
		t.Fatal(err)
	}
	manager, err := NewTUSManager(dir, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	_, payload, checksum := testPayload(64)
	id, err := manager.Create(ctx, int64(len(payload)), checksum, "image/jpeg")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Append(ctx, id, 0, payload); err != nil {
		t.Fatal(err)
	}
	count, err := manager.RecoverOrphans()
	if err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("live sessions and created files should be retained, reclaimed %d", count)
	}
}

func TestTUSHTTPUploadFlowAndErrorMapping(t *testing.T) {
	manager := newTestTUSManager(t, time.Hour)
	handler := TUSRouteHandler{Manager: manager}

	_, payload, checksum := testPayload(DefaultTUSChunkSize + 100)
	createBody := strings.NewReader(`{"size":` + strconv.Itoa(DefaultTUSChunkSize+100) + `,"checksum":"` + checksum + `","content_type":"image/jpeg"}`)
	create := httptest.NewRecorder()
	handler.ServeHTTP(create, httptest.NewRequest(http.MethodPost, "/uploads", createBody))
	if create.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", create.Code, create.Body.String())
	}
	var created tusCreateResponse
	if err := json.Unmarshal(create.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}

	// wrong offset on a fresh session → 409
	badChunk := base64.StdEncoding.EncodeToString(payload[:DefaultTUSChunkSize])
	wrongOffset := httptest.NewRecorder()
	handler.ServeHTTP(wrongOffset, httptest.NewRequest(http.MethodPost, "/uploads/"+created.ID+"/chunks", strings.NewReader(`{"offset":5,"data":"`+badChunk+`"}`)))
	if wrongOffset.Code != http.StatusConflict {
		t.Fatalf("offset mismatch status=%d want 409 body=%s", wrongOffset.Code, wrongOffset.Body.String())
	}

	// correct offset → 200
	appendBody := strings.NewReader(`{"offset":0,"data":"` + badChunk + `"}`)
	appendRec := httptest.NewRecorder()
	handler.ServeHTTP(appendRec, httptest.NewRequest(http.MethodPost, "/uploads/"+created.ID+"/chunks", appendBody))
	if appendRec.Code != http.StatusOK {
		t.Fatalf("append status=%d body=%s", appendRec.Code, appendRec.Body.String())
	}

	// query offset
	offsetRec := httptest.NewRecorder()
	handler.ServeHTTP(offsetRec, httptest.NewRequest(http.MethodGet, "/uploads/"+created.ID+"/offset", nil))
	if offsetRec.Code != http.StatusOK || !strings.Contains(offsetRec.Body.String(), strconv.Itoa(DefaultTUSChunkSize)) {
		t.Fatalf("offset status=%d body=%s", offsetRec.Code, offsetRec.Body.String())
	}

	// unknown id → 404
	missing := httptest.NewRecorder()
	handler.ServeHTTP(missing, httptest.NewRequest(http.MethodGet, "/uploads/nope/offset", nil))
	if missing.Code != http.StatusNotFound {
		t.Fatalf("unknown offset status=%d want 404", missing.Code)
	}

	// append rest + complete
	rest := base64.StdEncoding.EncodeToString(payload[DefaultTUSChunkSize:])
	appendRec = httptest.NewRecorder()
	handler.ServeHTTP(appendRec, httptest.NewRequest(http.MethodPost, "/uploads/"+created.ID+"/chunks", strings.NewReader(`{"offset":`+strconv.Itoa(DefaultTUSChunkSize)+`,"data":"`+rest+`"}`)))
	if appendRec.Code != http.StatusOK {
		t.Fatalf("final append status=%d body=%s", appendRec.Code, appendRec.Body.String())
	}
	completeRec := httptest.NewRecorder()
	handler.ServeHTTP(completeRec, httptest.NewRequest(http.MethodPost, "/uploads/"+created.ID+"/complete", nil))
	if completeRec.Code != http.StatusOK || !strings.Contains(completeRec.Body.String(), created.ID) {
		t.Fatalf("complete status=%d body=%s", completeRec.Code, completeRec.Body.String())
	}

	// complete a second time → 404 (session reclaimed)
	again := httptest.NewRecorder()
	handler.ServeHTTP(again, httptest.NewRequest(http.MethodPost, "/uploads/"+created.ID+"/complete", nil))
	if again.Code != http.StatusNotFound {
		t.Fatalf("re-complete status=%d want 404", again.Code)
	}
}

func TestTUSAbortUnknownReturnsNotFoundViaHTTP(t *testing.T) {
	manager := newTestTUSManager(t, time.Hour)
	handler := TUSRouteHandler{Manager: manager}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/uploads/unknown/abort", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("abort unknown status=%d want 404", rec.Code)
	}
}
