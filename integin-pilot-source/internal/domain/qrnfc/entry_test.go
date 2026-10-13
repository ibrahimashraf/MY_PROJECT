package qrnfc

import (
	"testing"
	"time"
)

func TestEntryValidate(t *testing.T) {
	now := time.Now()
	expiresAt := now.Add(24 * time.Hour)
	tests := []struct {
		name    string
		entry   Entry
		wantErr bool
	}{
		{
			name: "valid ACTIVE entry",
			entry: Entry{
				ID:             "entry_001",
				TenantID:       "tenant_1",
				OrganizationID: "org_1",
				EntitlementID:  "ent_001",
				AssetID:        "asset_001",
				EntryType:      EntryTypeQR,
				TokenDigest:    "abc123def456",
				TokenVersion:   1,
				Status:         EntryStatusActive,
				IssuedAt:       now,
				CreatedBy:      "user_1",
				CreatedAt:      now,
			},
			wantErr: false,
		},
		{
			name: "valid ACTIVE entry with expiry",
			entry: Entry{
				ID:             "entry_001",
				TenantID:       "tenant_1",
				OrganizationID: "org_1",
				EntitlementID:  "ent_001",
				AssetID:        "asset_001",
				EntryType:      EntryTypeNFC,
				TokenDigest:    "abc123def456",
				TokenVersion:   1,
				Status:         EntryStatusActive,
				IssuedAt:       now,
				ExpiresAt:      &expiresAt,
				CreatedBy:      "user_1",
				CreatedAt:      now,
			},
			wantErr: false,
		},
		{
			name: "missing entitlement_id",
			entry: Entry{
				ID:             "entry_001",
				TenantID:       "tenant_1",
				OrganizationID: "org_1",
				EntitlementID:  "",
				AssetID:        "asset_001",
				EntryType:      EntryTypeQR,
				TokenDigest:    "abc123def456",
				TokenVersion:   1,
				Status:         EntryStatusActive,
				IssuedAt:       now,
				CreatedBy:      "user_1",
				CreatedAt:      now,
			},
			wantErr: true,
		},
		{
			name: "unsupported entry_type",
			entry: Entry{
				ID:             "entry_001",
				TenantID:       "tenant_1",
				OrganizationID: "org_1",
				EntitlementID:  "ent_001",
				AssetID:        "asset_001",
				EntryType:      "INVALID",
				TokenDigest:    "abc123def456",
				TokenVersion:   1,
				Status:         EntryStatusActive,
				IssuedAt:       now,
				CreatedBy:      "user_1",
				CreatedAt:      now,
			},
			wantErr: true,
		},
		{
			name: "zero token_version",
			entry: Entry{
				ID:             "entry_001",
				TenantID:       "tenant_1",
				OrganizationID: "org_1",
				EntitlementID:  "ent_001",
				AssetID:        "asset_001",
				EntryType:      EntryTypeQR,
				TokenDigest:    "abc123def456",
				TokenVersion:   0,
				Status:         EntryStatusActive,
				IssuedAt:       now,
				CreatedBy:      "user_1",
				CreatedAt:      now,
			},
			wantErr: true,
		},
		{
			name: "ACTIVE with revoked_at",
			entry: Entry{
				ID:             "entry_001",
				TenantID:       "tenant_1",
				OrganizationID: "org_1",
				EntitlementID:  "ent_001",
				AssetID:        "asset_001",
				EntryType:      EntryTypeQR,
				TokenDigest:    "abc123def456",
				TokenVersion:   1,
				Status:         EntryStatusActive,
				IssuedAt:       now,
				RevokedAt:      &now,
				CreatedBy:      "user_1",
				CreatedAt:      now,
			},
			wantErr: true,
		},
		{
			name: "expires_at before issued_at",
			entry: Entry{
				ID:             "entry_001",
				TenantID:       "tenant_1",
				OrganizationID: "org_1",
				EntitlementID:  "ent_001",
				AssetID:        "asset_001",
				EntryType:      EntryTypeQR,
				TokenDigest:    "abc123def456",
				TokenVersion:   1,
				Status:         EntryStatusActive,
				IssuedAt:       now,
				ExpiresAt:      timePtr(now.Add(-1 * time.Hour)),
				CreatedBy:      "user_1",
				CreatedAt:      now,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.entry.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Entry.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCanTransitionTo(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name    string
		entry   Entry
		target  EntryStatus
		wantErr bool
	}{
		{
			name: "ACTIVE to REVOKED",
			entry: Entry{
				Status: EntryStatusActive,
			},
			target:  EntryStatusRevoked,
			wantErr: false,
		},
		{
			name: "ACTIVE to EXPIRED",
			entry: Entry{
				Status: EntryStatusActive,
			},
			target:  EntryStatusExpired,
			wantErr: false,
		},
		{
			name: "REVOKED to anything",
			entry: Entry{
				Status:    EntryStatusRevoked,
				RevokedAt: &now,
				RevokedBy: "user_1",
			},
			target:  EntryStatusActive,
			wantErr: true,
		},
		{
			name: "EXPIRED to anything",
			entry: Entry{
				Status: EntryStatusExpired,
			},
			target:  EntryStatusActive,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.entry.CanTransitionTo(tt.target)
			if (err != nil) != tt.wantErr {
				t.Errorf("CanTransitionTo() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGenerateToken(t *testing.T) {
	raw1, digest1, err := GenerateToken()
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}
	if len(raw1) != 43 {
		t.Errorf("token length = %d, want 43", len(raw1))
	}
	if len(digest1) != 64 {
		t.Errorf("digest length = %d, want 64", len(digest1))
	}
	raw2, digest2, err := GenerateToken()
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}
	if raw1 == raw2 {
		t.Error("GenerateToken() should produce different tokens")
	}
	if digest1 == digest2 {
		t.Error("GenerateToken() should produce different digests")
	}
}

func TestVerifyTokenDigest(t *testing.T) {
	rawToken, digest, err := GenerateToken()
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}
	if !VerifyTokenDigest(rawToken, digest) {
		t.Error("VerifyTokenDigest() should return true for valid token")
	}
	if VerifyTokenDigest("invalid-token", digest) {
		t.Error("VerifyTokenDigest() should return false for invalid token")
	}
	if VerifyTokenDigest(rawToken, "invalid-digest") {
		t.Error("VerifyTokenDigest() should return false for invalid digest")
	}
	if VerifyTokenDigest("short", digest) {
		t.Error("VerifyTokenDigest() should return false for short token")
	}
}

func TestRedactTokenForLog(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"long token", "ABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890abcd", "ABCD****abcd"},
		{"short token", "12345678", "****"},
		{"empty token", "", "****"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RedactTokenForLog(tt.input)
			if result != tt.expected {
				t.Errorf("RedactTokenForLog(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func timePtr(t time.Time) *time.Time {
	return &t
}

func TestEntryLogValidate(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name    string
		log     EntryLog
		wantErr bool
	}{
		{
			name: "valid SUCCESS log",
			log: EntryLog{
				ID:             "log_001",
				TenantID:       "tenant_1",
				OrganizationID: "org_1",
				EntryID:        "entry_001",
				EntryType:      EntryTypeQR,
				AccessedAt:     now,
				AccessorID:     "user_1",
				AccessorIP:     "192.168.1.1",
				Outcome:        AccessOutcomeSuccess,
			},
			wantErr: false,
		},
		{
			name: "missing entry_id",
			log: EntryLog{
				ID:             "log_001",
				TenantID:       "tenant_1",
				OrganizationID: "org_1",
				EntryID:        "",
				EntryType:      EntryTypeQR,
				AccessedAt:     now,
				AccessorID:     "user_1",
				Outcome:        AccessOutcomeSuccess,
			},
			wantErr: true,
		},
		{
			name: "unsupported outcome",
			log: EntryLog{
				ID:             "log_001",
				TenantID:       "tenant_1",
				OrganizationID: "org_1",
				EntryID:        "entry_001",
				EntryType:      EntryTypeQR,
				AccessedAt:     now,
				AccessorID:     "user_1",
				Outcome:        "INVALID",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.log.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("EntryLog.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
