package syncstate

import "testing"

func TestDurableContractsRejectIncompleteTenantOrTrustRecords(t *testing.T) {
	if err := ValidateDeviceRecord(DeviceRecord{DeviceID: "device-1", TenantID: "tenant-1", OrganizationID: "org-1", UserID: "user-1", KeyID: "key-1", PublicKey: []byte("public"), AuthorityEpoch: 1}); err != nil {
		t.Fatal(err)
	}
	if err := ValidateDeviceRecord(DeviceRecord{DeviceID: "device-1", TenantID: "tenant-1"}); err == nil {
		t.Fatal("expected incomplete device record to fail")
	}
	if err := ValidateReceipt(Receipt{TransactionID: "tx-1", TenantID: "tenant-1", OrganizationID: "org-1", DeviceID: "device-1", UserID: "user-1", SequenceNumber: 1, Operation: "InspectionSubmitted", EntityID: "inspection-1", PayloadHash: "hash", Outcome: "APPLIED"}); err != nil {
		t.Fatal(err)
	}
}
