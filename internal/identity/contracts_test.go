// integin identity foundation tests: local identity keys and membership-denial reasons remain independent of token claims.
package identity

import "testing"

func TestValidatePrincipalKeyRejectsBlankValues(t *testing.T) {
	for _, principal := range []PrincipalKey{{}, {Issuer: "issuer"}, {Subject: "subject"}} {
		if err := ValidatePrincipalKey(principal); err != ErrUnknownSubject {
			t.Fatalf("expected unknown-subject error for %+v, got %v", principal, err)
		}
	}
}

func TestValidatePrincipalKeyAcceptsIssuerSubjectOnly(t *testing.T) {
	if err := ValidatePrincipalKey(PrincipalKey{Issuer: "https://issuer.example/realms/pilot", Subject: "subject-001"}); err != nil {
		t.Fatalf("validate issuer-subject key: %v", err)
	}
}
