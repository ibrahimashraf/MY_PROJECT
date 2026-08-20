package main

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"integin/internal/domain/sync"
)

func TestPublicMatrixErrorStagesAreClosedAndWrapStable(t *testing.T) {
	seen := make(map[string]struct{}, len(matrixErrorStages))
	seenSentinels := make(map[error]struct{}, len(matrixErrorStages))
	for _, stage := range matrixErrorStages {
		if !isPublicMatrixErrorCode(stage.code) {
			t.Fatalf("stage code %q is not public", stage.code)
		}
		if _, duplicate := seen[stage.code]; duplicate {
			t.Fatalf("duplicate stage code %q", stage.code)
		}
		seen[stage.code] = struct{}{}
		if _, duplicate := seenSentinels[stage.err]; duplicate {
			t.Fatalf("duplicate stage sentinel for %q", stage.code)
		}
		seenSentinels[stage.err] = struct{}{}
		if got := matrixErrorCode(fmt.Errorf("wrapped: %w", stage.err)); got != stage.code {
			t.Fatalf("wrapped stage %q mapped to %q", stage.code, got)
		}
		for _, other := range matrixErrorStages {
			if other.code != stage.code && matrixErrorCode(fmt.Errorf("wrapped: %w", stage.err)) == other.code {
				t.Fatalf("stage %q cross-matched %q", stage.code, other.code)
			}
		}
	}
	if got := matrixErrorCode(errors.New("unclassified local failure")); got != "MATRIX_CLASSIFICATION_FAILED" {
		t.Fatalf("unclassified error mapped to %q", got)
	}
	if _, found := seen["MATRIX_CLASSIFICATION_FAILED"]; found {
		t.Fatal("classification fallback must not be a named stage")
	}
	if len(publicMatrixCodes) != len(seen)+1 {
		t.Fatalf("public code set has %d entries, stages plus fallback have %d", len(publicMatrixCodes), len(seen)+1)
	}
	for code := range publicMatrixCodes {
		if code != "MATRIX_CLASSIFICATION_FAILED" {
			if _, found := seen[code]; !found {
				t.Fatalf("public code %q has no named stage", code)
			}
		}
	}
}

func TestLoadFixtureInvalidInputCarriesFixtureLoadSentinel(t *testing.T) {
	_, err := loadFixture(filepath.Join(t.TempDir(), "missing-public-fixture.json"))
	if !errors.Is(err, errFixtureLoad) {
		t.Fatalf("invalid fixture error does not carry errFixtureLoad: %v", err)
	}
}

func TestWritePublicErrorCodeRejectsUnknownAndPreexistingTargets(t *testing.T) {
	dir := t.TempDir()
	invalidPath := filepath.Join(dir, "invalid-code.txt")
	if err := writePublicErrorCode(invalidPath, "NOT_A_PUBLIC_CODE"); err == nil {
		t.Fatal("unknown public code was accepted")
	}
	if _, err := os.Stat(invalidPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("unknown public code created an artifact: %v", err)
	}

	preexistingPath := filepath.Join(dir, "preexisting-code.txt")
	if err := os.WriteFile(preexistingPath, []byte("existing"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := writePublicErrorCode(preexistingPath, "FIXTURE_LOAD_FAILED"); err == nil {
		t.Fatal("preexisting public code artifact was overwritten")
	}
	content, err := os.ReadFile(preexistingPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "existing" {
		t.Fatalf("preexisting artifact content changed to %q", content)
	}
}

func TestExpectStatusPreservesAssignedPublicCaseSentinel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))
	defer server.Close()

	for _, statusErr := range []error{
		errValidProofStatus,
		errReplayStatus,
		errSignatureInvalidStatus,
		errExpiredStatus,
		errAuthorityMismatchStatus,
		errKeyUnknownStatus,
		errPackageHashStatus,
	} {
		err := expectStatus(server.Client(), server.URL, sync.DeviceProof{}, http.StatusOK, statusErr)
		if !errors.Is(err, statusErr) {
			t.Fatalf("status mismatch did not preserve assigned sentinel: %v", err)
		}
	}
}
