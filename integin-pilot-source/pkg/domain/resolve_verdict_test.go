package domain

import (
	"errors"
	"testing"

	"integin/pkg/rulesengine"
)

var errDBDown = errors.New("db down")

func testPassport() *UniversalAssetPassport {
	return &UniversalAssetPassport{
		AssetDID:      "did:integin:asset:7f8a9e",
		Manufacturer:  "Liebherr",
		ChassisSerial: "SER-0001",
		CurrentStatus: AssetStatusOperational,
	}
}

func TestResolveAssetDID(t *testing.T) {
	passport := testPassport()
	tests := []struct {
		name    string
		raw     string
		lookup  func(identifier string) (*UniversalAssetPassport, error)
		wantErr error
		wantDID string
	}{
		{
			name: "success",
			raw:  "did:integin:asset:7f8a9e",
			lookup: func(id string) (*UniversalAssetPassport, error) {
				if id != "7f8a9e" {
					t.Fatalf("lookup got identifier %q", id)
				}
				return passport, nil
			},
			wantDID: "did:integin:asset:7f8a9e",
		},
		{
			name:    "invalid prefix",
			raw:     "foo:bar:asset:xyz",
			lookup:  passthroughLookup,
			wantErr: ErrInvalidDIDPrefix,
		},
		{
			name:    "malformed",
			raw:     "did:integin:asset",
			lookup:  passthroughLookup,
			wantErr: ErrMalformedDID,
		},
		{
			name:    "non-asset type",
			raw:     "did:integin:standard:ASME_B30_5_2024",
			lookup:  passthroughLookup,
			wantErr: ErrNotAssetDID,
		},
		{
			name: "not found",
			raw:  "did:integin:asset:missing",
			lookup: func(id string) (*UniversalAssetPassport, error) {
				return nil, nil
			},
			wantErr: ErrAssetNotFound,
		},
		{
			name: "lookup error propagated",
			raw:  "did:integin:asset:7f8a9e",
			lookup: func(id string) (*UniversalAssetPassport, error) {
				return nil, errDBDown
			},
			wantErr: errDBDown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveAssetDID(tt.raw, tt.lookup)
			if tt.wantErr == nil {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if got == nil {
					t.Fatal("expected passport, got nil")
				}
				if got.AssetDID != tt.wantDID {
					t.Fatalf("AssetDID = %q, want %q", got.AssetDID, tt.wantDID)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error %v, got nil", tt.wantErr)
			}
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func passthroughLookup(id string) (*UniversalAssetPassport, error) {
	return testPassport(), nil
}

func TestApplyProofLoadVerdict(t *testing.T) {
	t.Run("nil passport returns error", func(t *testing.T) {
		applied, err := ApplyProofLoadVerdict(nil, rulesengine.SeverityCriticalQuarantine, "boom")
		if applied {
			t.Fatal("applied = true, want false")
		}
		if !errors.Is(err, ErrNilPassport) {
			t.Fatalf("err = %v, want %v", err, ErrNilPassport)
		}
	})

	t.Run("critical quarantines", func(t *testing.T) {
		passport := testPassport()
		applied, err := ApplyProofLoadVerdict(passport, rulesengine.SeverityCriticalQuarantine, "crane boom rent")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !applied {
			t.Fatal("applied = false, want true")
		}
		if passport.CurrentStatus != AssetStatusQuarantined {
			t.Fatalf("CurrentStatus = %q, want %q", passport.CurrentStatus, AssetStatusQuarantined)
		}
		if got, ok := passport.CustomAttributes["quarantine_reason"].(string); !ok || got != "crane boom rent" {
			t.Fatalf("quarantine_reason = %v, want %q", passport.CustomAttributes["quarantine_reason"], "crane boom rent")
		}
	})

	t.Run("warning does not quarantine", func(t *testing.T) {
		passport := testPassport()
		applied, err := ApplyProofLoadVerdict(passport, rulesengine.SeverityWarning, "minor deviation")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if applied {
			t.Fatal("applied = true, want false")
		}
		if passport.CurrentStatus != AssetStatusOperational {
			t.Fatalf("CurrentStatus = %q, want %q", passport.CurrentStatus, AssetStatusOperational)
		}
	})

	t.Run("empty severity does not quarantine", func(t *testing.T) {
		passport := testPassport()
		applied, err := ApplyProofLoadVerdict(passport, rulesengine.Severity(""), "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if applied {
			t.Fatal("applied = true, want false")
		}
	})
}
