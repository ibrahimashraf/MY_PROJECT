package main

import (
	"context"
	"crypto/ed25519"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"time"

	domainsync "integin/internal/domain/sync"
	"integin/internal/domain/workpackage"
	"integin/internal/workpackagepg"
)

func policyInspectionID(deviceID string) string {
	return "integin-policy-inspection-" + deviceID
}

// policySeed first reuses the existing disposable device and authority fixture
// flow, then reads that local fixture only to bind a real approved package to
// the newly created disposable device.
func policySeed() error {
	if err := seed(); err != nil {
		return err
	}
	databaseURL, _, _, fixturePath, err := localConfig()
	if err != nil {
		return err
	}
	content, err := os.ReadFile(fixturePath)
	if err != nil {
		return err
	}
	var value fixture
	if err := json.Unmarshal(content, &value); err != nil {
		return err
	}
	database, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return err
	}
	defer database.Close()
	return seedPolicyBinding(context.Background(), database, value.TenantID, value.Organization, value.DeviceID)
}

// seedPolicyBinding persists one approved package and one current, expiring
// disposable assignment through repository methods. It creates no acceptance
// data and prints no fixture/key material.
func seedPolicyBinding(ctx context.Context, database *sql.DB, tenantID, organizationID, deviceID string) error {
	repository, err := workpackagepg.NewRepository(database)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	packageValue, err := (workpackage.Package{
		ID:              "integin-policy-package-" + deviceID,
		TenantID:        tenantID,
		OrganizationID:  organizationID,
		TemplateCode:    "pilot-policy-matrix",
		TemplateVersion: 1,
		PackageVersion:  1,
		SchemaVersion:   1,
		State:           workpackage.PublicationApproved,
		Sections: []workpackage.Section{{
			ID:    "main",
			Title: "Main",
			Fields: []workpackage.FieldDefinition{
				{ID: "condition", Prompt: "Condition", Type: workpackage.FieldPassFailNA, Required: true},
				{ID: "capacity", Prompt: "Capacity", Type: workpackage.FieldNumber, Required: true},
			},
		}},
	}).WithComputedHash()
	if err != nil {
		return err
	}
	if err := repository.SaveApproved(ctx, packageValue, now); err != nil {
		return err
	}
	return repository.AssignApproved(ctx, workpackage.Assignment{
		TenantID:       tenantID,
		OrganizationID: organizationID,
		InspectionID:   policyInspectionID(deviceID),
		DeviceID:       deviceID,
		PackageID:      packageValue.ID,
		PackageVersion: packageValue.PackageVersion,
		AuthorityEpoch: 1,
		AssignedAt:     now,
		ExpiresAt:      now.Add(30 * time.Minute),
	}, now)
}

// policyExercise reads only the disposable fixture and repository identity,
// submits a valid signed inspection once, replays it once, and submits one
// invalid-signature transaction. Its output is bounded to outcome categories.
func policyExercise() error {
	databaseURL, _, _, fixturePath, err := localConfig()
	if err != nil {
		return err
	}
	database, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return err
	}
	defer database.Close()

	content, err := os.ReadFile(fixturePath)
	if err != nil {
		return err
	}
	var value fixture
	if err := json.Unmarshal(content, &value); err != nil {
		return err
	}
	privateKey, err := base64.StdEncoding.DecodeString(value.PrivateKey)
	if err != nil || len(privateKey) != ed25519.PrivateKeySize {
		return errors.New("fixture has invalid private key")
	}

	repository, err := workpackagepg.NewRepository(database)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	assignment, err := repository.GetCurrentAssignment(context.Background(), value.TenantID, value.Organization, policyInspectionID(value.DeviceID), value.DeviceID, now)
	if err != nil {
		return err
	}
	packageValue, err := repository.GetApproved(context.Background(), value.TenantID, value.Organization, assignment.PackageID, assignment.PackageVersion)
	if err != nil {
		return err
	}
	serverURL, err := explicitPilotServerURL(os.Getenv("INTEGIN_SERVER_URL"))
	if err != nil {
		return err
	}
	payload, err := json.Marshal(map[string]any{
		"inspection_id":               policyInspectionID(value.DeviceID),
		"work_package_id":             packageValue.ID,
		"work_package_version":        packageValue.PackageVersion,
		"work_package_hash":           packageValue.PackageHash,
		"work_package_field_sequence": []string{"condition", "capacity"},
		"findings": []map[string]any{
			{"item_id": "condition", "response": "pass"},
			{"item_id": "capacity", "response": "12.5"},
		},
	})
	if err != nil {
		return err
	}

	client := &http.Client{Timeout: 15 * time.Second}
	prefix := "integin-policy-" + value.DeviceID
	valid, err := policySignedTransaction(prefix+"-valid", 1, value, privateKey, policyInspectionID(value.DeviceID), payload)
	if err != nil {
		return err
	}
	if err := expectSync(client, serverURL, valid, "APPLIED"); err != nil {
		return err
	}
	if err := printPolicyMatrixState(context.Background(), database, value.TenantID, value.DeviceID, "after_valid"); err != nil {
		return err
	}
	if err := expectSync(client, serverURL, valid, "DUPLICATE"); err != nil {
		return err
	}
	if err := printPolicyMatrixState(context.Background(), database, value.TenantID, value.DeviceID, "after_duplicate"); err != nil {
		return err
	}
	securityFailure, err := policySignedTransaction(prefix+"-security", 2, value, privateKey, policyInspectionID(value.DeviceID), payload)
	if err != nil {
		return err
	}
	securityFailure.Signature = "invalid-signature"
	if err := expectSync(client, serverURL, securityFailure, "SECURITY_FAILURE"); err != nil {
		return err
	}
	if err := printPolicyMatrixState(context.Background(), database, value.TenantID, value.DeviceID, "after_signature_failure"); err != nil {
		return err
	}
	return nil
}

func policySignedTransaction(id string, sequence uint64, value fixture, privateKey ed25519.PrivateKey, inspectionID string, payload []byte) (domainsync.Transaction, error) {
	transaction := domainsync.NewTransaction(id, value.TenantID, value.DeviceID, value.UserID, sequence, "InspectionSubmitted", payload)
	transaction.OrganizationID = value.Organization
	transaction.EntityID = inspectionID
	transaction.AuthorityID = value.AuthorityID
	transaction.AuthorityEpoch = 1
	transaction.CapturedAt = time.Now().UTC().Truncate(time.Microsecond)
	return domainsync.SignTransactionEd25519(transaction, privateKey, value.KeyID)
}
