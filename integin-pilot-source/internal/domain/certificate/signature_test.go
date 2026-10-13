package certificate

import (
	"strings"
	"testing"
	"time"
)

const (
	testImageSHA    = "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08"
	testSnapshotSHA = "5feceb66ffc86f38d952786c6d696c79c2dbc239dd4e91b46729d73a27fb57e9"
)

func approveTestCertificate(t *testing.T) Certificate {
	t.Helper()
	certificate := newTestCertificate(t)
	if err := certificate.SubmitForApproval("creator-1"); err != nil {
		t.Fatal(err)
	}
	if err := certificate.Approve("approver-1", nil); err != nil {
		t.Fatal(err)
	}
	return certificate
}

func validClientEvent() SignatureEvent {
	return SignatureEvent{
		SignerID:         "client-rep-1",
		SignerName:       "Sahil Singh",
		Capacity:         CapacityClient,
		StatementVersion: "client_ack_v1",
		ImageSHA256Hex:   testImageSHA,
		ImageBytes:       48210,
		ImageEvidenceID:  "evidence-signature-1",
		SnapshotSHA256:   testSnapshotSHA,
	}
}

func TestSignWithEventSealsServerDerivedFields(t *testing.T) {
	certificate := approveTestCertificate(t)
	before := time.Now().UTC()
	if err := certificate.SignWithEvent(validClientEvent()); err != nil {
		t.Fatal(err)
	}
	if certificate.Status() != Signed {
		t.Fatalf("expected SIGNED, got %s", certificate.Status())
	}
	signatures := certificate.Signatures()
	if len(signatures) != 1 {
		t.Fatalf("expected one signature, got %d", len(signatures))
	}
	sealed := signatures[0]
	if sealed.CertificateNumber != "CERT-001" || sealed.Revision != 1 {
		t.Fatalf("event not bound to certificate identity: %+v", sealed)
	}
	if sealed.SignedAt.Before(before) || sealed.SignedAt.Location() != time.UTC {
		t.Fatalf("server timestamp not sealed: %s", sealed.SignedAt)
	}
	if sealed.ImageSHA256Hex != testImageSHA || sealed.SnapshotSHA256 != testSnapshotSHA {
		t.Fatal("digests not recorded")
	}
}

func TestSignWithEventRejectsHollowEvents(t *testing.T) {
	cases := map[string]func(*SignatureEvent){
		"blank signer":    func(e *SignatureEvent) { e.SignerID = " " },
		"blank name":      func(e *SignatureEvent) { e.SignerName = "" },
		"bad capacity":    func(e *SignatureEvent) { e.Capacity = "witness" },
		"blank statement": func(e *SignatureEvent) { e.StatementVersion = "" },
		"short image sha": func(e *SignatureEvent) { e.ImageSHA256Hex = "abc" },
		"nonhex image":    func(e *SignatureEvent) { e.ImageSHA256Hex = strings.Repeat("zz", 32) },
		"zero bytes":      func(e *SignatureEvent) { e.ImageBytes = 0 },
		"oversize image":  func(e *SignatureEvent) { e.ImageBytes = MaxSignatureImageBytes + 1 },
		"blank evidence":  func(e *SignatureEvent) { e.ImageEvidenceID = " " },
		"blank snapshot":  func(e *SignatureEvent) { e.SnapshotSHA256 = "" },
	}
	for name, mutate := range cases {
		certificate := approveTestCertificate(t)
		event := validClientEvent()
		mutate(&event)
		if err := certificate.SignWithEvent(event); err == nil {
			t.Fatalf("%s should be rejected", name)
		}
		if certificate.Status() != Approved || len(certificate.Signatures()) != 0 {
			t.Fatalf("%s must not mutate certificate", name)
		}
	}
}

func TestSignWithEventRejectsInspectorCapacity(t *testing.T) {
	certificate := approveTestCertificate(t)
	event := validClientEvent()
	event.Capacity = CapacityInspector
	if err := certificate.SignWithEvent(event); err == nil {
		t.Fatal("inspector capacity signing should be rejected")
	}
	event = validClientEvent()
	event.SignerID = "inspector-1"
	if err := certificate.SignWithEvent(event); err == nil {
		t.Fatal("assigned inspector signing should be rejected")
	}
}

func TestAttestRecordsInspectorFindingAttestation(t *testing.T) {
	certificate := approveTestCertificate(t)
	if err := certificate.Attest("inspector-1", "Inspector I", "Company Appointed Examiner", "inspector_attest_v1", testSnapshotSHA); err != nil {
		t.Fatal(err)
	}
	if certificate.Status() != Approved {
		t.Fatalf("attestation must not change status, got %s", certificate.Status())
	}
	signatures := certificate.Signatures()
	if len(signatures) != 1 || signatures[0].Capacity != CapacityInspector {
		t.Fatalf("attestation not recorded: %+v", signatures)
	}
	if signatures[0].SignedAt.Location() != time.UTC {
		t.Fatal("attestation timestamp must be UTC")
	}
}

func TestAttestRejectsNonInspectorAndBlankBasis(t *testing.T) {
	certificate := approveTestCertificate(t)
	if err := certificate.Attest("approver-1", "Approver", "Company Appointed Examiner", "inspector_attest_v1", testSnapshotSHA); err == nil {
		t.Fatal("non-inspector attestation should be rejected")
	}
	if err := certificate.Attest("inspector-1", "Inspector I", "", "inspector_attest_v1", testSnapshotSHA); err == nil {
		t.Fatal("blank qualification basis should be rejected")
	}
	if len(certificate.Signatures()) != 0 {
		t.Fatal("failed attestation must not record")
	}
}

func TestWaiveSignatureSealsWaiverAndAdvances(t *testing.T) {
	certificate := approveTestCertificate(t)
	if err := certificate.WaiveSignature("authority-1", "client unreachable on site, office confirmed by phone", CapacityClient, ""); err != nil {
		t.Fatal(err)
	}
	if certificate.Status() != Signed {
		t.Fatalf("expected SIGNED, got %s", certificate.Status())
	}
	waiver := certificate.SignWaiver()
	if waiver == nil {
		t.Fatal("waiver not recorded")
	}
	if waiver.GrantedBy != "authority-1" || waiver.Capacity != CapacityClient {
		t.Fatalf("waiver miscaptured: %+v", waiver)
	}
	if waiver.CertificateNumber != "CERT-001" || waiver.Revision != 1 || waiver.WaivedAt.Location() != time.UTC {
		t.Fatalf("waiver not sealed: %+v", waiver)
	}
	if len(certificate.Signatures()) != 0 {
		t.Fatal("waiver must not fabricate a signature event")
	}
}

func TestWaiveSignatureRejectsHollowWaivers(t *testing.T) {
	for name, tc := range map[string][4]string{
		"blank granter":   {" ", "reason", CapacityClient, ""},
		"blank reason":    {"authority-1", "", CapacityClient, ""},
		"oversize reason": {"authority-1", strings.Repeat("r", MaxTextFieldChars+1), CapacityClient, ""},
		"bad capacity":    {"authority-1", "reason", CapacityInspector, ""},
		"self authorized": {"authority-1", "reason", CapacityClient, "authority-1"},
	} {
		certificate := approveTestCertificate(t)
		if err := certificate.WaiveSignature(tc[0], tc[1], tc[2], tc[3]); err == nil {
			t.Fatalf("%s should be rejected", name)
		}
		if certificate.Status() != Approved || certificate.SignWaiver() != nil {
			t.Fatalf("%s must not mutate certificate", name)
		}
	}
	draft := newTestCertificate(t)
	if err := draft.WaiveSignature("authority-1", "reason", CapacityClient, ""); err == nil {
		t.Fatal("waiver from draft should be rejected")
	}
}

func TestWaiveSignatureHybridRule(t *testing.T) {
	inspector := approveTestCertificate(t)
	if err := inspector.WaiveSignature("inspector-1", "client off-site", CapacityClient, ""); err == nil {
		t.Fatal("inspector waiver without office authorizer should be rejected")
	}
	if err := inspector.WaiveSignature("inspector-1", "client off-site", CapacityClient, "office-1"); err != nil {
		t.Fatal(err)
	}
	if waiver := inspector.SignWaiver(); waiver == nil || waiver.AuthorizedBy != "office-1" {
		t.Fatalf("authorizer not recorded: %+v", waiver)
	}
}
