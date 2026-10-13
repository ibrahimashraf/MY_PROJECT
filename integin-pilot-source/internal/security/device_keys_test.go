package security

import "testing"

func TestDeviceKeyPairSignsAndVerifies(t *testing.T) {
	publicKey, privateKey, err := GenerateDeviceKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	message := []byte("tenant-1|org-1|tx-1")
	signature, err := SignDeviceMutation(privateKey, message)
	if err != nil {
		t.Fatal(err)
	}
	if !VerifyDeviceMutation(publicKey, message, signature) {
		t.Fatal("expected signature to verify")
	}
	if VerifyDeviceMutation(publicKey, []byte("tampered"), signature) {
		t.Fatal("expected tampered message to fail")
	}
	if DeviceKeyID(publicKey) == "" {
		t.Fatal("expected key id")
	}
}
