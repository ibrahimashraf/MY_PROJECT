package server

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"integin/internal/oidcauth"
	"integin/internal/storage"
)

type tusValidatorStub struct{ err error }

func (s tusValidatorStub) Validate(_ context.Context, _ string) (oidcauth.Principal, error) {
	return oidcauth.Principal{Issuer: "issuer", Subject: "subject"}, s.err
}

func TestTUSRoutesRejectUnauthenticatedThroughMux(t *testing.T) {
	manager, err := storage.NewTUSManager(t.TempDir(), 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	mux := NewMux(Dependencies{TUSHandler: storage.TUSRouteHandler{Manager: manager}, Validator: tusValidatorStub{}})
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/uploads", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("no-bearer create status=%d want %d body=%s", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}

	rejecting := NewMux(Dependencies{TUSHandler: storage.TUSRouteHandler{Manager: manager}, Validator: tusValidatorStub{err: errors.New("rejected")}})
	request := httptest.NewRequest(http.MethodPost, "/uploads", nil)
	request.Header.Set("Authorization", "Bearer invalid-token")
	rec = httptest.NewRecorder()
	rejecting.ServeHTTP(rec, request)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("invalid-bearer status=%d want %d body=%s", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
}

func TestTUSRoutesFailClosedWithoutValidator(t *testing.T) {
	manager, err := storage.NewTUSManager(t.TempDir(), 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	mux := NewMux(Dependencies{TUSHandler: storage.TUSRouteHandler{Manager: manager}})
	rec := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/uploads", nil)
	request.Header.Set("Authorization", "Bearer any-token")
	mux.ServeHTTP(rec, request)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("uploads without a validator must fail closed, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestTUSAuthenticatedUploadFlowThroughMux(t *testing.T) {
	manager, err := storage.NewTUSManager(t.TempDir(), 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	mux := NewMux(Dependencies{TUSHandler: storage.TUSRouteHandler{Manager: manager}, Validator: tusValidatorStub{}})

	payload := make([]byte, 3000)
	for i := range payload {
		payload[i] = byte(i * 13)
	}
	digest := sha256.Sum256(payload)
	checksum := hex.EncodeToString(digest[:])

	authorized := func(request *http.Request) *http.Request {
		request.Header.Set("Authorization", "Bearer valid-token")
		return request
	}
	create := httptest.NewRecorder()
	mux.ServeHTTP(create, authorized(httptest.NewRequest(http.MethodPost, "/uploads", strings.NewReader(`{"size":3000,"checksum":"`+checksum+`","content_type":"video/mp4"}`))))
	if create.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", create.Code, create.Body.String())
	}
	var created struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(create.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.ID == "" {
		t.Fatal("create returned no id")
	}

	appendChunk := func(offset int, chunk []byte) {
		rec := httptest.NewRecorder()
		body := `{"offset":` + strconv.Itoa(offset) + `,"data":"` + base64.StdEncoding.EncodeToString(chunk) + `"}`
		mux.ServeHTTP(rec, authorized(httptest.NewRequest(http.MethodPost, "/uploads/"+created.ID+"/chunks", strings.NewReader(body))))
		if rec.Code != http.StatusOK {
			t.Fatalf("append at offset %d status=%d body=%s", offset, rec.Code, rec.Body.String())
		}
	}
	appendChunk(0, payload[:1500])
	appendChunk(1500, payload[1500:])

	offsetRec := httptest.NewRecorder()
	mux.ServeHTTP(offsetRec, authorized(httptest.NewRequest(http.MethodGet, "/uploads/"+created.ID+"/offset", nil)))
	if offsetRec.Code != http.StatusOK || !strings.Contains(offsetRec.Body.String(), `"offset":3000`) {
		t.Fatalf("offset status=%d body=%s", offsetRec.Code, offsetRec.Body.String())
	}

	completeRec := httptest.NewRecorder()
	mux.ServeHTTP(completeRec, authorized(httptest.NewRequest(http.MethodPost, "/uploads/"+created.ID+"/complete", nil)))
	if completeRec.Code != http.StatusOK || !strings.Contains(completeRec.Body.String(), created.ID) {
		t.Fatalf("complete status=%d body=%s", completeRec.Code, completeRec.Body.String())
	}
}
