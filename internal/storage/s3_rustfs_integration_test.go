package storage

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"os"
	"testing"
	"time"
)

func TestRustFSS3Contract(t *testing.T) {
	if os.Getenv("INTEGIN_RUSTFS_INTEGRATION") != "1" {
		t.Skip("set INTEGIN_RUSTFS_INTEGRATION=1 to run against a configured RustFS endpoint")
	}
	endpoint := os.Getenv("INTEGIN_S3_ENDPOINT")
	bucket := os.Getenv("INTEGIN_S3_BUCKET")
	accessKey := os.Getenv("INTEGIN_S3_ACCESS_KEY")
	secretKey := os.Getenv("INTEGIN_S3_SECRET_KEY")
	region := os.Getenv("INTEGIN_S3_REGION")
	if region == "" {
		region = "us-east-1"
	}

	signer := NewAWSSigV4Signer(accessKey, secretKey, region, "s3")
	store, err := NewS3Store(endpoint, bucket, http.DefaultClient, signer)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := store.Health(ctx); err != nil {
		t.Fatalf("signed health check: %v", err)
	}

	plaintext := []byte("integin RustFS plaintext integrity vector")
	ciphertext := []byte("integin RustFS encrypted evidence bytes v1")
	plaintextDigest := hexDigest(plaintext)
	ciphertextDigest := hexDigest(ciphertext)
	key := "integration/rustfs-contract-" + time.Now().UTC().Format("20060102T150405.000000000") + ".bin"
	object := Object{
		Key:         key,
		ContentType: "application/octet-stream",
		Data:        ciphertext,
		Metadata: map[string]string{
			"plaintext-sha256":  plaintextDigest,
			"ciphertext-sha256": ciphertextDigest,
		},
	}
	if err := store.Put(ctx, object); err != nil {
		t.Fatalf("signed put: %v", err)
	}
	defer func() {
		if err := store.Delete(context.Background(), key); err != nil {
			t.Logf("cleanup delete %q: %v", key, err)
		}
	}()

	got, err := store.Get(ctx, key)
	if err != nil {
		t.Fatalf("signed get: %v", err)
	}
	if !bytes.Equal(got.Data, ciphertext) {
		t.Fatalf("downloaded ciphertext mismatch: got %x want %x", got.Data, ciphertext)
	}

	headURL, err := store.objectURL(key)
	if err != nil {
		t.Fatal(err)
	}
	headRequest, err := http.NewRequestWithContext(ctx, http.MethodHead, headURL.String(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := signer(headRequest); err != nil {
		t.Fatal(err)
	}
	headResponse, err := http.DefaultClient.Do(headRequest)
	if err != nil {
		t.Fatalf("signed head: %v", err)
	}
	defer headResponse.Body.Close()
	if headResponse.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(headResponse.Body)
		t.Fatalf("signed head status %d: %s", headResponse.StatusCode, body)
	}
	if gotDigest := headResponse.Header.Get("X-Amz-Meta-Ciphertext-Sha256"); gotDigest != ciphertextDigest {
		t.Fatalf("ciphertext digest metadata = %q, want %q", gotDigest, ciphertextDigest)
	}
	if gotDigest := headResponse.Header.Get("X-Amz-Meta-Plaintext-Sha256"); gotDigest != plaintextDigest {
		t.Fatalf("plaintext digest metadata = %q, want %q", gotDigest, plaintextDigest)
	}

	if err := store.Put(ctx, object); err != nil {
		t.Fatalf("duplicate put must remain idempotent at storage layer: %v", err)
	}
	if err := store.Delete(ctx, key); err != nil {
		t.Fatalf("signed delete: %v", err)
	}
	if _, err := store.Get(ctx, key); err == nil {
		t.Fatal("get after delete unexpectedly succeeded")
	}
}

func TestRustFSRestartDurability(t *testing.T) {
	if os.Getenv("INTEGIN_RUSTFS_INTEGRATION") != "1" {
		t.Skip("set INTEGIN_RUSTFS_INTEGRATION=1 to run against a configured RustFS endpoint")
	}
	stage := os.Getenv("INTEGIN_RUSTFS_RESTART_STAGE")
	if stage == "" {
		t.Skip("set INTEGIN_RUSTFS_RESTART_STAGE=seed or verify for the controlled restart drill")
	}
	endpoint := os.Getenv("INTEGIN_S3_ENDPOINT")
	bucket := os.Getenv("INTEGIN_S3_BUCKET")
	accessKey := os.Getenv("INTEGIN_S3_ACCESS_KEY")
	secretKey := os.Getenv("INTEGIN_S3_SECRET_KEY")
	region := os.Getenv("INTEGIN_S3_REGION")
	if region == "" {
		region = "us-east-1"
	}
	store, err := NewS3Store(endpoint, bucket, http.DefaultClient, NewAWSSigV4Signer(accessKey, secretKey, region, "s3"))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	key := "integration/rustfs-restart-durability.bin"
	payload := []byte("integin RustFS restart durability encrypted evidence vector")
	digest := hexDigest(payload)

	switch stage {
	case "seed":
		if err := store.Put(ctx, Object{
			Key:         key,
			ContentType: "application/octet-stream",
			Data:        payload,
			Metadata: map[string]string{
				"ciphertext-sha256": digest,
			},
		}); err != nil {
			t.Fatalf("seed durable object: %v", err)
		}
	case "verify":
		got, err := store.Get(ctx, key)
		if err != nil {
			t.Fatalf("get seeded object after RustFS restart: %v", err)
		}
		if !bytes.Equal(got.Data, payload) {
			t.Fatalf("post-restart payload mismatch: got %x want %x", got.Data, payload)
		}
		if err := store.Delete(ctx, key); err != nil {
			t.Fatalf("delete verified restart object: %v", err)
		}
	default:
		t.Fatalf("unsupported INTEGIN_RUSTFS_RESTART_STAGE %q", stage)
	}
}

func hexDigest(value []byte) string {
	digest := sha256.Sum256(value)
	return hex.EncodeToString(digest[:])
}
