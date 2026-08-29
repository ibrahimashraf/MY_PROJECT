package storage

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) Do(request *http.Request) (*http.Response, error) { return f(request) }

func TestInMemoryStorageHandlesObjectsWithoutAliasing(t *testing.T) {
	store := NewInMemoryStore()
	ctx := context.Background()
	data := []byte("evidence")
	object := Object{Key: "evidence/tenant-1/photo.jpg", ContentType: "image/jpeg", Data: data, Metadata: map[string]string{"tenant": "tenant-1"}}
	if err := store.Put(ctx, object); err != nil {
		t.Fatal(err)
	}
	data[0] = 'X'
	object.Metadata["tenant"] = "changed"
	loaded, err := store.Get(ctx, object.Key)
	if err != nil {
		t.Fatal(err)
	}
	if string(loaded.Data) != "evidence" || loaded.Metadata["tenant"] != "tenant-1" {
		t.Fatalf("storage aliasing detected: %#v", loaded)
	}
	if err := store.Delete(ctx, object.Key); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(ctx, object.Key); err == nil {
		t.Fatal("deleted object should not be found")
	}
}

func TestMinIOStoreBuildsSignedObjectRequest(t *testing.T) {
	var method, requestURL, signature string
	client := roundTripFunc(func(request *http.Request) (*http.Response, error) {
		method, requestURL = request.Method, request.URL.String()
		signature = request.Header.Get("Authorization")
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("")), Header: make(http.Header)}, nil
	})
	store, err := NewMinIOStore("http://minio.local:9000", "integin", client, func(request *http.Request) error { request.Header.Set("Authorization", "signed"); return nil })
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Put(context.Background(), Object{Key: "tenant-1/certificate.pdf", ContentType: "application/pdf", Data: []byte("pdf")}); err != nil {
		t.Fatal(err)
	}
	if method != http.MethodPut || !strings.Contains(requestURL, "/integin/tenant-1/certificate.pdf") || signature != "signed" {
		t.Fatalf("unexpected MinIO request: %s %s %s", method, requestURL, signature)
	}
	if _, err := store.Get(context.Background(), "../secret"); err == nil {
		t.Fatal("path traversal key should be rejected")
	}
}

func TestAWSSigV4SignerBindsPayloadDigest(t *testing.T) {
	var requestHeaders http.Header
	client := roundTripFunc(func(request *http.Request) (*http.Response, error) {
		requestHeaders = request.Header.Clone()
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("")), Header: make(http.Header)}, nil
	})
	store, err := NewMinIOStore("http://minio.local:9000", "integin", client, NewAWSSigV4Signer("access", "secret", "us-east-1", "s3"))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Put(context.Background(), Object{Key: "tenant-1/evidence.bin", ContentType: "application/octet-stream", Data: []byte("evidence")}); err != nil {
		t.Fatal(err)
	}
	if requestHeaders.Get("X-Amz-Content-Sha256") != sha256Hex([]byte("evidence")) {
		t.Fatalf("payload digest was not bound to the request: %q", requestHeaders.Get("X-Amz-Content-Sha256"))
	}
	if requestHeaders.Get("X-Amz-Date") == "" || !strings.HasPrefix(requestHeaders.Get("Authorization"), "AWS4-HMAC-SHA256 Credential=access/") {
		t.Fatalf("Signature V4 headers missing: %#v", requestHeaders)
	}
}
func TestS3StoreGetReturnsCiphertextAndMetadata(t *testing.T) {
	var method, signature string
	client := roundTripFunc(func(request *http.Request) (*http.Response, error) {
		method = request.Method
		signature = request.Header.Get("Authorization")
		header := make(http.Header)
		header.Set("Content-Type", "application/octet-stream")
		header.Set("X-Amz-Meta-Tenant_ID", "tenant-1")
		header.Set("X-Amz-Meta-Inspection_ID", "inspection-1")
		header.Set("X-Amz-Meta-Ciphertext_SHA256", "abc123")
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("ciphertext")), Header: header}, nil
	})
	store, err := NewS3Store("http://rustfs.local:9000", "integin-evidence", client, func(request *http.Request) error {
		request.Header.Set("Authorization", "signed")
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	object, err := store.Get(context.Background(), "tenant-1/org-1/evidence/evidence-1")
	if err != nil {
		t.Fatal(err)
	}
	if method != http.MethodGet || signature != "signed" {
		t.Fatalf("unexpected Get request: method=%q signature=%q", method, signature)
	}
	if object.ContentType != "application/octet-stream" || string(object.Data) != "ciphertext" {
		t.Fatalf("unexpected object body: %#v", object)
	}
	if object.Metadata["tenant_id"] != "tenant-1" || object.Metadata["inspection_id"] != "inspection-1" || object.Metadata["ciphertext_sha256"] != "abc123" {
		t.Fatalf("metadata was not returned or normalized: %#v", object.Metadata)
	}
}
