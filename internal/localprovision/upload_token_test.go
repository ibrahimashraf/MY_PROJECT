package localprovision

import (
	"testing"
	"time"
)

func TestUploadTokenRoundTrip(t *testing.T) {
	expires := time.Now().UTC().Add(30 * time.Minute)
	token, err := MintUploadToken("secret", "device-1", "auth-1", 3, expires)
	if err != nil {
		t.Fatal(err)
	}
	claims, err := ValidateUploadToken("secret", token, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if claims.DeviceID != "device-1" || claims.AuthorityID != "auth-1" || claims.Epoch != 3 {
		t.Fatalf("unexpected claims: %+v", claims)
	}
	if claims.ExpiresAt.Unix() != expires.Unix() {
		t.Fatalf("expiry not preserved: %s vs %s", claims.ExpiresAt, expires)
	}
}

func TestUploadTokenRejectsTamperingAndExpiry(t *testing.T) {
	now := time.Now().UTC()
	token, err := MintUploadToken("secret", "device-1", "auth-1", 1, now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ValidateUploadToken("wrong-secret", token, now); err == nil {
		t.Fatal("wrong secret must be rejected")
	}
	tampered := token[:len(token)-2] + "xx"
	if _, err := ValidateUploadToken("secret", tampered, now); err == nil {
		t.Fatal("tampered signature must be rejected")
	}
	if _, err := ValidateUploadToken("secret", "v1.not-base64!!.also-not", now); err == nil {
		t.Fatal("malformed token must be rejected")
	}
	expired, err := MintUploadToken("secret", "device-1", "auth-1", 1, now.Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ValidateUploadToken("secret", expired, now); err == nil {
		t.Fatal("expired token must be rejected")
	}
	if _, err := MintUploadToken("", "device-1", "auth-1", 1, now); err == nil {
		t.Fatal("empty secret must be refused at mint")
	}
}
