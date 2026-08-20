package main

import (
	"errors"
	"fmt"
	"testing"
)

func TestPublicMatrixErrorStagesAreClosedAndWrapStable(t *testing.T) {
	seen := make(map[string]struct{}, len(matrixErrorStages))
	for _, stage := range matrixErrorStages {
		if !isPublicMatrixErrorCode(stage.code) {
			t.Fatalf("stage code %q is not public", stage.code)
		}
		if _, duplicate := seen[stage.code]; duplicate {
			t.Fatalf("duplicate stage code %q", stage.code)
		}
		seen[stage.code] = struct{}{}
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
}
