package manifestreceipts

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const lifecycleRunID = "0123456789abcdef0123456789abcdef"

func TestFinalizeV2CompleteInventoryWritesPublicTerminalArtifact(t *testing.T) {
	directory := t.TempDir()
	for _, c := range ExpectedV2Cases() {
		if err := os.WriteFile(filepath.Join(directory, "receipt-v2-"+string(c)+"-"+lifecycleRunID+".json"), []byte(`{}`), 0o600); err != nil {
			t.Fatalf("write receipt placeholder: %v", err)
		}
	}
	result, err := FinalizeV2(lifecycleRunID, directory, func() time.Time { return time.Date(2026, time.August, 19, 9, 0, 0, 0, time.UTC) })
	if err != nil {
		t.Fatalf("finalize complete inventory: %v", err)
	}
	if result.Status != V2FinalizationComplete || result.EmittedCaseCount != len(ExpectedV2Cases()) || result.BridgeFailurePresent || result.ReasonCode != "" {
		t.Fatalf("unexpected finalization: %#v", result)
	}
	content, err := os.ReadFile(filepath.Join(directory, v2FinalizationArtifactName(lifecycleRunID)))
	if err != nil {
		t.Fatalf("read finalization artifact: %v", err)
	}
	var artifact map[string]any
	if err := json.Unmarshal(content, &artifact); err != nil {
		t.Fatalf("decode finalization artifact: %v", err)
	}
	if artifact["status"] != string(V2FinalizationComplete) || artifact["artifact"] != "bridge_finalization" || artifact["bridge_failure_present"] != false {
		t.Fatalf("unexpected public finalization artifact: %#v", artifact)
	}
}

func TestFinalizeV2MissingCaseIsIncompleteWithoutRawErrors(t *testing.T) {
	directory := t.TempDir()
	cases := ExpectedV2Cases()
	for _, c := range cases[1:] {
		if err := os.WriteFile(filepath.Join(directory, "receipt-v2-"+string(c)+"-"+lifecycleRunID+".json"), []byte(`{}`), 0o600); err != nil {
			t.Fatalf("write receipt placeholder: %v", err)
		}
	}
	result, err := FinalizeV2(lifecycleRunID, directory, time.Now)
	if err != nil {
		t.Fatalf("finalize incomplete inventory: %v", err)
	}
	if result.Status != V2FinalizationIncomplete || result.ReasonCode != V2FailureMissingExpectedCase || result.EmittedCaseCount != len(cases)-1 {
		t.Fatalf("unexpected incomplete finalization: %#v", result)
	}
}

func TestEmitWithFailureRecordsOneRedactedDuplicateFailure(t *testing.T) {
	directory := t.TempDir()
	writer, err := NewV2Writer(lifecycleRunID, directory, func() time.Time { return time.Date(2026, time.August, 19, 10, 0, 0, 0, time.UTC) })
	if err != nil {
		t.Fatalf("new writer: %v", err)
	}
	observation := V2Observation{Case: CaseFieldBinding, EventSource: V2EventSourceFieldBinding, ObservedOutcome: "verified_cached", Transport: V2Transport{Kind: V2TransportFieldBinding}, ReplayState: V2ReplayState{Scope: V2ReplayScopeNotApplicable}}
	if err := writer.EmitWithFailure(observation); err != nil {
		t.Fatalf("emit first observation: %v", err)
	}
	if err := writer.EmitWithFailure(observation); !errors.Is(err, ErrDuplicateReceipt) {
		t.Fatalf("duplicate error = %v, want duplicate receipt", err)
	}
	content, err := os.ReadFile(filepath.Join(directory, v2FailureArtifactName(lifecycleRunID)))
	if err != nil {
		t.Fatalf("read bridge failure artifact: %v", err)
	}
	if strings.Contains(string(content), "ErrDuplicateReceipt") || strings.Contains(string(content), "manifest v2") {
		t.Fatalf("failure artifact exposed raw error: %s", content)
	}
	var artifact map[string]any
	if err := json.Unmarshal(content, &artifact); err != nil {
		t.Fatalf("decode bridge failure artifact: %v", err)
	}
	if artifact["failure_code"] != string(V2FailureDuplicateCase) || artifact["case"] != string(CaseFieldBinding) {
		t.Fatalf("unexpected bridge failure artifact: %#v", artifact)
	}
}

func TestFinalizeV2FailureOrUnexpectedArtifactFailsClosed(t *testing.T) {
	for _, testCase := range []struct {
		name  string
		setup func(t *testing.T, directory string)
		code  V2FailureCode
	}{
		{name: "bridge failure", code: V2FailureBridgeFailure, setup: func(t *testing.T, directory string) {
			t.Helper()
			for _, c := range ExpectedV2Cases() {
				if err := os.WriteFile(filepath.Join(directory, "receipt-v2-"+string(c)+"-"+lifecycleRunID+".json"), []byte(`{}`), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.WriteFile(filepath.Join(directory, v2FailureArtifactName(lifecycleRunID)), []byte(`{"contract_version":"2"}`), 0o600); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "unexpected artifact", code: V2FailureUnexpectedArtifact, setup: func(t *testing.T, directory string) {
			t.Helper()
			if err := os.WriteFile(filepath.Join(directory, "unexpected.txt"), []byte("not public evidence"), 0o600); err != nil {
				t.Fatal(err)
			}
		}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			directory := t.TempDir()
			testCase.setup(t, directory)
			result, err := FinalizeV2(lifecycleRunID, directory, time.Now)
			if err != nil {
				t.Fatalf("finalize: %v", err)
			}
			if result.Status != V2FinalizationFailed || result.ReasonCode != testCase.code {
				t.Fatalf("unexpected failed finalization: %#v", result)
			}
		})
	}
}

func TestFinalizeV2RejectsDuplicateFinalization(t *testing.T) {
	directory := t.TempDir()
	if _, err := FinalizeV2(lifecycleRunID, directory, time.Now); err != nil {
		t.Fatalf("first finalization: %v", err)
	}
	if _, err := FinalizeV2(lifecycleRunID, directory, time.Now); !errors.Is(err, ErrDuplicateReceipt) {
		t.Fatalf("duplicate finalization error = %v, want duplicate receipt", err)
	}
}

func TestFinalizeV2RejectsUnsafeDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(path, []byte("not a receipt directory"), 0o600); err != nil {
		t.Fatalf("write unsafe directory stand-in: %v", err)
	}
	if _, err := FinalizeV2(lifecycleRunID, path, time.Now); !errors.Is(err, ErrUnsafeDirectory) {
		t.Fatalf("unsafe directory error = %v, want ErrUnsafeDirectory", err)
	}
}
