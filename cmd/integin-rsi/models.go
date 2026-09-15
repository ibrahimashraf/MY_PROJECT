package main

// MutationRequest is a single closed-loop Red-Green-Refactor iteration:
// a proposed diff over target files, verified against a Go package.
type MutationRequest struct {
	TargetFiles []string `json:"target_files"`
	Patch       string   `json:"patch"`
	TestPackage string   `json:"test_package"`
	MaxAttempts int      `json:"max_attempts"`
}

// IterationResult reports the outcome of one mutation verification cycle.
type IterationResult struct {
	Success             bool   `json:"success"`
	Attempt             int    `json:"attempt"`
	VetOutput           string `json:"vet_output,omitempty"`
	TestOutput          string `json:"test_output,omitempty"`
	ErrorReason         string `json:"error_reason,omitempty"`
	ExecutionDurationMs int64  `json:"execution_duration_ms"`
}
