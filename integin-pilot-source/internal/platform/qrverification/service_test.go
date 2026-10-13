package qrverification

import "testing"

func TestQRPublicAndTechnicalVerificationBoundaries(t *testing.T) {
	verifier := New(2)
	if err := verifier.Register(Certificate{Token: "qr-1", Environment: "TESTING", Status: "ISSUED", Number: "CERT-1", Revision: 1, AssetType: "offshore_container", Manufacturer: "Acme", Model: "CCU", SerialNumber: "SN-1", ManufactureDate: "2020-01-01", SWL: "10t", CCUStamp: "V", Issuer: "issuer-1", InspectionDate: "2026-08-13", ExpiryDate: "2027-08-13", Findings: []string{"internal finding"}, Photos: []string{"private.jpg"}, ClientName: "Secret Client", Site: "Secret Site", Inspector: "secret-user", InternalID: "internal-1"}); err != nil {
		t.Fatal(err)
	}
	publicResult := verifier.PublicVerify("qr-1", "ip-1")
	if publicResult.StatusCode != 200 || publicResult.Public == nil {
		t.Fatalf("unexpected public result: %#v", publicResult)
	}
	if publicResult.Public.SerialNumber != "SN-1" || !publicResult.Public.TestCertificate {
		t.Fatal("public allow-list or test marker incorrect")
	}
	if publicResult.Public.AssetType == "" {
		t.Fatal("asset type should be public")
	}
	technical := verifier.TechnicalVerify("qr-1", "engineer-1", func(userID string, c Certificate) bool { return userID == "engineer-1" })
	if technical.StatusCode != 200 || technical.Technical == nil || len(technical.Technical.Findings) != 1 {
		t.Fatalf("unexpected technical result: %#v", technical)
	}
	unauthorized := verifier.TechnicalVerify("qr-1", "client-1", func(string, Certificate) bool { return false })
	if unauthorized.StatusCode != 403 {
		t.Fatalf("expected 403, got %#v", unauthorized)
	}
}

func TestQRRateLimitAndMissingCertificate(t *testing.T) {
	verifier := New(1)
	_ = verifier.Register(Certificate{Token: "qr-1", Status: "ISSUED", Number: "CERT-1"})
	if result := verifier.PublicVerify("qr-1", "ip-1"); result.StatusCode != 200 {
		t.Fatal(result)
	}
	if result := verifier.PublicVerify("qr-1", "ip-1"); result.StatusCode != 429 {
		t.Fatalf("expected rate limit: %#v", result)
	}
	if result := verifier.PublicVerify("missing", "ip-2"); result.StatusCode != 404 {
		t.Fatalf("expected 404: %#v", result)
	}
}
