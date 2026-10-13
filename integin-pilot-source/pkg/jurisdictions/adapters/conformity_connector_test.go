package adapters

import (
	"errors"
	"testing"
)

func TestSASO_PCoCPass(t *testing.T) {
	res, err := (ConformityConnector{}).VerifyAccreditation(
		"SA", "PCoC-501234", "SABER certificate PCoC-501234 issued under SASO conformity assessment",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Status != StatusPass {
		t.Fatalf("status = %v, want PASS: %+v", res.Status, res.Findings)
	}
	if res.Scheme != "SASO SABER PCoC" || res.Accreditor != "SASO" {
		t.Fatalf("result = %+v", res)
	}
	for _, f := range res.Findings {
		if f.Status != StatusPass {
			t.Fatalf("expected only pass findings: %+v", res.Findings)
		}
	}
}

func conformityResultHas(t *testing.T, res ConformityResult, want ComplianceStatus, field string) {
	t.Helper()
	if res.Status != want {
		t.Fatalf("status = %v, want %v: %+v", res.Status, want, res.Findings)
	}
	for _, f := range res.Findings {
		if f.Field == field {
			return
		}
	}
	t.Fatalf("no finding on field %q: %+v", field, res.Findings)
}

func TestSASO_SCoCPass(t *testing.T) {
	res, err := (ConformityConnector{}).VerifyAccreditation(
		"SA", "SCoC-90124", "SCoC-90124 shipment certificate verified active under SASO SABER",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Status != StatusPass || res.Scheme != "SASO SABER SCoC" {
		t.Fatalf("result = %+v", res)
	}
}

func TestSASO_InvalidFormatFailsClosed(t *testing.T) {
	res, err := (ConformityConnector{}).VerifyAccreditation(
		"SA", "XYZ-123", "excellent certification document",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	conformityResultHas(t, res, StatusFail, "certificate_reference")
}

func TestSASO_RevokedBodyFailsClosed(t *testing.T) {
	res, err := (ConformityConnector{}).VerifyAccreditation(
		"SA", "PCoC-501234", "SABER PCoC-501234 REVOKED by SASO",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	conformityResultHas(t, res, StatusFail, "certificate_status")
}

func TestSASO_MissingAccreditorFailsClosed(t *testing.T) {
	res, err := (ConformityConnector{}).VerifyAccreditation(
		"SA", "PCoC-501234", "self-issued certificate, no conformity body",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	conformityResultHas(t, res, StatusFail, "accreditation")
}

func TestSASO_EmptyCertRefFailsClosed(t *testing.T) {
	res, err := (ConformityConnector{}).VerifyAccreditation("SA", "  ", "SASO body")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	conformityResultHas(t, res, StatusFail, "certificate_reference")
}

func TestUAE_EIACPass(t *testing.T) {
	res, err := (ConformityConnector{}).VerifyAccreditation(
		"AE", "EIAC-ISL-1932", "EIAC accreditation scope EIAC-ISL-1932 currently active",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Status != StatusPass || res.Scheme != "EIAC Accreditation Scope" || res.Accreditor != "EIAC" {
		t.Fatalf("result = %+v", res)
	}
}

func TestUAE_DACPass(t *testing.T) {
	res, err := (ConformityConnector{}).VerifyAccreditation(
		"AE", "DAC-LAB-2155", "DAR accredited laboratory per DAC scope DAC-LAB-2155",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Status != StatusPass || res.Scheme != "DAC Accreditation Scope" {
		t.Fatalf("result = %+v", res)
	}
}

func TestUAE_InvalidSchemeFailsClosed(t *testing.T) {
	res, err := (ConformityConnector{}).VerifyAccreditation(
		"AE", "PCoC-8391", "SASO body text irrelevant here",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	conformityResultHas(t, res, StatusFail, "certificate_reference")
}

func TestUAE_BodyMissingAccreditorFailsClosed(t *testing.T) {
	res, err := (ConformityConnector{}).VerifyAccreditation(
		"AE", "EIAC-ISL-1932", "certificate issued, accreditor unnamed",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	conformityResultHas(t, res, StatusFail, "accreditation")
}

func TestCaseInsensitiveJurisdiction(t *testing.T) {
	res, err := (ConformityConnector{}).VerifyAccreditation(
		"sa", "PCoC-501234", "SASO certificate",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.JurisdictionISO2 != "SA" || res.Status != StatusPass {
		t.Fatalf("result = %+v", res)
	}
}

func TestUnsupportedJurisdiction(t *testing.T) {
	_, err := (ConformityConnector{}).VerifyAccreditation("FR", "PCoC-1", "body")
	if !errors.Is(err, ErrUnsupportedJurisdiction) {
		t.Fatalf("err = %v, want %v", err, ErrUnsupportedJurisdiction)
	}
}
