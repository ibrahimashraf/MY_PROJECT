// Package envcompat bridges the INTEGIN_ → INTEGIN_ environment rename.
//
// New configuration uses INTEGIN_ variables. Every INTEGIN_ variable ever read
// by this codebase keeps working: when INTEGIN_X is set and INTEGIN_X is not,
// Mirror copies it over before any other boot code reads the environment.
// When both are set, the explicit INTEGIN_ value wins (current behavior is
// preserved exactly). Call Mirror once at process start.
package envcompat

import (
	"os"
	"strings"
)

const (
	NewPrefix = "INTEGIN_"
	OldPrefix = "integin_"
)

// Mirror copies INTEGIN_* over unset INTEGIN_* counterparts.
func Mirror() {
	for _, kv := range os.Environ() {
		name, _, found := strings.Cut(kv, "=")
		if !found || !strings.HasPrefix(name, NewPrefix) {
			continue
		}
		old := OldPrefix + strings.TrimPrefix(name, NewPrefix)
		if _, exists := os.LookupEnv(old); !exists {
			_ = os.Setenv(old, os.Getenv(name))
		}
	}
}
