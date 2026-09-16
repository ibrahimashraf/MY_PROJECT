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

func TestTUSResumeAfterRestart(t *testing.T) {
	dir := t.TempDir()
	manager, err := NewTUSManager(dir, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
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
	// Kill-simulated restart: a brand-new manager over the same directory must
	// re-attach the session from its sidecar instead of deleting the chunk file.
	recovered, err := NewTUSManager(dir, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	session, err := recovered.Offset(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if session.Offset != DefaultTUSChunkSize*2 || session.Size != int64(len(payload)) || session.Checksum != checksum {
		t.Fatalf("restart did not resume the declared session state: %#v", session)
	}
	if _, err := recovered.Append(ctx, id, session.Offset, payload[DefaultTUSChunkSize*2:]); err != nil {
		t.Fatal(err)
	}
	object, err := recovered.Complete(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if string(object.Data) != string(payload) {
		t.Fatal("resumed upload did not reassemble the declared payload")
	}
	if _, err := os.Stat(filepath.Join(dir, id+".json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("sidecar should be removed after completion, stat err=%v", err)
	}
}

func TestTUSRecoverOrphansDeletesUnrecoverableFiles(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"deadbeefdeadbeefdeadbeefdeadbeef", "cafebabecafebabecafebabecafebabe.part"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("partial chunk bytes"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := NewTUSManager(dir, 0, 0); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("all unrecoverable orphan files should be deleted on restart, found %d", len(entries))
	}
}

func TestTUSRecoverOrphansDeletesCorruptSidecar(t *testing.T) {
	dir := t.TempDir()
	manager, err := NewTUSManager(dir, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	id := "deadbeefdeadbeefdeadbeefdeadbeef"
	if err := os.WriteFile(filepath.Join(dir, id), []byte("partial bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, id+".json"), []byte("{not-json"), 0o600); err != nil {
		t.Fatal(err)
	}
	resumed, deleted, err := manager.RecoverOrphans()
	if err != nil {
		t.Fatal(err)
	}
	if resumed != 0 || deleted != 1 {
		t.Fatalf("corrupt sidecar must be deleted and never resumed: resumed=%d deleted=%d", resumed, deleted)
	}
	for _, name := range []string{id, id + ".json"} {
		if _, err := os.Stat(filepath.Join(dir, name)); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("corrupt session file %s should be deleted, stat err=%v", name, err)
		}
	}
}

func TestTUSRecoverOrphansDeletesTruncatedSidecarAndTmp(t *testing.T) {
	dir := t.TempDir()
	manager, err := NewTUSManager(dir, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	id := "deadbeefdeadbeefdeadbeefdeadbeef"
	if err := os.WriteFile(filepath.Join(dir, id), []byte("partial chunk bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, id+".json"), []byte(`{"id":"deadbeef`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "cafebabecafebabecafebabecafebabe.tmp"), []byte(`{"id":"cafe`), 0o600); err != nil {
		t.Fatal(err)
	}
	resumed, deleted, err := manager.RecoverOrphans()
	if err != nil {
		t.Fatal(err)
	}
	if resumed != 0 || deleted != 2 {
		t.Fatalf("truncated sidecar and stray tmp must both be deleted: resumed=%d deleted=%d", resumed, deleted)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("no temporary sidecar files may remain after recovery, found %d", len(entries))
	}
}

func TestTUSResumeAfterCrashBeforeSidecarWrite(t *testing.T) {
	dir := t.TempDir()
	manager, err := NewTUSManager(dir, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	_, payload, checksum := testPayload(DefaultTUSChunkSize * 4)
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
	// Crash landing between the chunk write and its sidecar write: the next
	// chunk's bytes are on disk while the sidecar still describes the previous
	// committed offset.
	file, err := os.OpenFile(filepath.Join(dir, id), os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write(payload[DefaultTUSChunkSize*2 : DefaultTUSChunkSize*3]); err != nil {
		if closeErr := file.Close(); closeErr != nil {
			t.Fatalf("direct chunk write: %v (close: %v)", err, closeErr)
		}
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	// Kill-simulated restart over the same directory.
	recovered, err := NewTUSManager(dir, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	session, err := recovered.Offset(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if session.Offset != DefaultTUSChunkSize*3 {
		t.Fatalf("recovered offset must be the actual chunk-file size, got %d", session.Offset)
	}
	if _, err := recovered.Append(ctx, id, session.Offset, payload[DefaultTUSChunkSize*3:]); err != nil {
		t.Fatal(err)
	}
	object, err := recovered.Complete(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if string(object.Data) != string(payload) {
		t.Fatal("resumed upload did not reassemble the declared payload")
	}
}

func TestTUSRecoverOrphansDeletesOffsetMismatch(t *testing.T) {
	dir := t.TempDir()
	manager, err := NewTUSManager(dir, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	id := "cafebabecafebabecafebabecafebabe"
	if err := os.WriteFile(filepath.Join(dir, id), []byte("0123456789"), 0o600); err != nil {
		t.Fatal(err)
	}
	sidecar, err := json.Marshal(tusSidecar{ID: id, Size: 64, Checksum: strings.Repeat("0", 64), Offset: 32, ContentType: "image/png"})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, id+".json"), sidecar, 0o600); err != nil {
		t.Fatal(err)
	}
	resumed, deleted, err := manager.RecoverOrphans()
	if err != nil {
		t.Fatal(err)
	}
	if resumed != 0 || deleted != 1 {
		t.Fatalf("offset mismatch must be deleted, not resumed: resumed=%d deleted=%d", resumed, deleted)
	}
	if _, err := os.Stat(filepath.Join(dir, id)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("mismatched chunk file should be deleted, stat err=%v", err)
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
	resumed, deleted, err := manager.RecoverOrphans()
	if err != nil {
		t.Fatal(err)
	}
	if resumed != 0 || deleted != 0 {
		t.Fatalf("live sessions and their sidecars must be retained: resumed=%d deleted=%d", resumed, deleted)
	}
}

func TestTUSHTTPUploadFlowAndErrorMapping(t *testing.T) {
	manager := newTestTUSManager(t, time.Hour)
	store := NewInMemoryStore()
	handler := TUSRouteHandler{Manager: manager, Store: store}

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

func TestTUSRouteHandlerRejectsOversizedAndUnknownFields(t *testing.T) {
	manager := newTestTUSManager(t, time.Hour)
	store := NewInMemoryStore()
	handler := TUSRouteHandler{Manager: manager, Store: store}

	// Unknown field in create
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/uploads", strings.NewReader(`{"size":100,"checksum":"`+strings.Repeat("a", 64)+`","content_type":"image/jpeg","unknown":"field"}`)))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("unknown field status=%d want 400", rec.Code)
	}

	// Size <= 0
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/uploads", strings.NewReader(`{"size":0,"checksum":"`+strings.Repeat("a", 64)+`","content_type":"image/jpeg"}`)))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("zero size status=%d want 400", rec.Code)
	}

	// Size > MaxUploadSize (100 MiB)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/uploads", strings.NewReader(`{"size":`+strconv.FormatInt(MaxUploadSize+1, 10)+`,"checksum":"`+strings.Repeat("a", 64)+`","content_type":"image/jpeg"}`)))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("oversized status=%d want 400", rec.Code)
	}

	// Dangerous content type (e.g. application/x-executable)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/uploads", strings.NewReader(`{"size":100,"checksum":"`+strings.Repeat("a", 64)+`","content_type":"application/x-executable"}`)))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("dangerous content type status=%d want 400", rec.Code)
	}
}

func TestTUSRouteHandlerEnforcesChunkRemainingBytes(t *testing.T) {
	manager := newTestTUSManager(t, time.Hour)
	store := NewInMemoryStore()
	handler := TUSRouteHandler{Manager: manager, Store: store}

	ctx := context.Background()
	_, payload, checksum := testPayload(100)
	id, err := manager.Create(ctx, 100, checksum, "image/jpeg")
	if err != nil {
		t.Fatal(err)
	}

	// Try to append 150 bytes when remaining is 100
	oversizedChunk := base64.StdEncoding.EncodeToString(make([]byte, 150))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/uploads/"+id+"/chunks", strings.NewReader(`{"offset":0,"data":"`+oversizedChunk+`"}`)))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("chunk exceeding remaining status=%d want 400", rec.Code)
	}

	// Append valid payload
	validChunk := base64.StdEncoding.EncodeToString(payload)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/uploads/"+id+"/chunks", strings.NewReader(`{"offset":0,"data":"`+validChunk+`"}`)))
	if rec.Code != http.StatusOK {
		t.Fatalf("valid chunk append status=%d want 200", rec.Code)
	}

	// Complete should persist to store
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/uploads/"+id+"/complete", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("complete status=%d want 200", rec.Code)
	}

	obj, err := store.Get(ctx, id)
	if err != nil {
		t.Fatalf("completed object was not persisted in Store: %v", err)
	}
	if obj.Key != id || len(obj.Data) != 100 {
		t.Fatalf("unexpected stored object: %#v", obj)
	}
}
