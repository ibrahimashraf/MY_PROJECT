package manifestreceiptbridge

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"integin/internal/manifestreceipts"
	"integin/internal/packagemanifestapi"
)

const observerRunID = "abcdef0123456789abcdef0123456789"

func TestHTTPObserverEmitsOnlyClosedValidProofMapping(t *testing.T) {
	directory := t.TempDir()
	writer, err := manifestreceipts.NewV2Writer(observerRunID, directory, func() time.Time { return time.Date(2026, time.August, 19, 12, 0, 0, 0, time.UTC) })
	if err != nil {
		t.Fatal(err)
	}
	before, after := int64(4), int64(5)
	observer := NewHTTPObserver(writer)
	observer.Observe(context.TODO(), packagemanifestapi.Observation{Outcome: "manifest_issued", ReasonCode: "valid_proof", HTTPStatus: 200, ReplayAccepted: true, ReplayBefore: &before, ReplayAfter: &after})
	if _, err := os.Stat(filepath.Join(directory, "receipt-v2-valid_proof-"+observerRunID+".json")); err != nil {
		t.Fatalf("valid proof receipt missing: %v", err)
	}

	observer.Observe(context.TODO(), packagemanifestapi.Observation{Outcome: "manifest_issued", ReasonCode: "valid_proof", HTTPStatus: 200, ReplayAccepted: true, ManifestID: "manifest-must-not-appear-in-failure"})
	failurePath := filepath.Join(directory, "bridge-failure-v2-"+observerRunID+".json")
	if _, err := os.Stat(failurePath); err != nil {
		t.Fatalf("missing snapshot did not produce bridge failure: %v", err)
	}
	failure, err := os.ReadFile(failurePath)
	if err != nil {
		t.Fatalf("read bridge failure artifact: %v", err)
	}
	if strings.Contains(string(failure), "manifest-must-not-appear-in-failure") {
		t.Fatalf("bridge failure artifact exposed raw manifest identifier: %s", failure)
	}
}

func TestHTTPObserverRejectsNearMatchesAndDiagnostics(t *testing.T) {
	directory := t.TempDir()
	writer, err := manifestreceipts.NewV2Writer(observerRunID, directory, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	observer := NewHTTPObserver(writer)
	for _, observation := range []packagemanifestapi.Observation{
		{Outcome: "request_rejected", ReasonCode: "invalid_request", HTTPStatus: 400},
		{Outcome: "proof_rejected", ReasonCode: "signature_invalid", HTTPStatus: 403},
		{Outcome: "manifest_issued", ReasonCode: "valid_proof", HTTPStatus: 201},
	} {
		observer.Observe(context.TODO(), observation)
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("diagnostic/near-match observations wrote artifacts: %#v", entries)
	}
}

func TestHTTPObserverAcceptsObservedIssuerExpiryStatus(t *testing.T) {
	observation := packagemanifestapi.Observation{Outcome: "proof_rejected", ReasonCode: "expired", HTTPStatus: 403}
	receipt, ok := mapHTTPObservation(observation)
	if !ok || receipt.Case != manifestreceipts.CaseExpired || receipt.Transport.HTTPStatus == nil || *receipt.Transport.HTTPStatus != 403 {
		t.Fatalf("issuer-expiry mapping = %#v, mapped=%v", receipt, ok)
	}
}
