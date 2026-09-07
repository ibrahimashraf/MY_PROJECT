package audit

import "testing"

func TestAuditLedgerIsTenantScopedAndAppendOnly(t *testing.T) {
	ledger := NewLedger()
	if err := ledger.Append(Record{ID: "audit-1", TenantID: "tenant-1", Environment: "LIVE", ActorType: "USER", ActorID: "user-1", Action: "CertificateIssued", ResourceType: "certificate", ResourceID: "cert-1", Outcome: "SUCCESS"}); err != nil {
		t.Fatal(err)
	}
	if err := ledger.Append(Record{ID: "audit-2", TenantID: "tenant-2", Environment: "LIVE", ActorType: "USER", ActorID: "user-2", Action: "CertificateIssued", ResourceType: "certificate", ResourceID: "cert-2", Outcome: "SUCCESS"}); err != nil {
		t.Fatal(err)
	}
	records := ledger.List("tenant-1")
	if len(records) != 1 || records[0].TenantID != "tenant-1" {
		t.Fatalf("unexpected tenant audit records: %#v", records)
	}
	if len(ledger.List("")) != 2 {
		t.Fatal("append-only ledger lost records")
	}
}
