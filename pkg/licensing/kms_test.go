package licensing_test

import (
	"testing"

	"integin/pkg/licensing"
)

func TestOpenBaoKMSClient_Init(t *testing.T) {
	client, err := licensing.NewOpenBaoKMSClient("http://127.0.0.1:8200", "test-token")
	if err != nil {
		t.Fatalf("NewOpenBaoKMSClient failed: %v", err)
	}
	if client.Client() == nil {
		t.Fatal("expected non-nil vault client")
	}

	_, err = licensing.NewOpenBaoKMSClient("", "token")
	if err == nil {
		t.Fatal("expected error on empty address")
	}
}
