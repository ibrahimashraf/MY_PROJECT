// Package env provides clamped environment-variable overrides for timeouts,
// TTLs, and retry budgets. Defaults are preserved when the variable is
// absent or out of range.
package env

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Seconds reads key as integer seconds, clamped to [min, max].
// It returns fallback when unset, unparsable, or out of range.
func Seconds(key string, fallback time.Duration, min, max int) time.Duration {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	s, err := strconv.Atoi(v)
	if err != nil || s < min || s > max {
		return fallback
	}
	return time.Duration(s) * time.Second
}

// Int reads key as an integer, clamped to [min, max].
// It returns fallback when unset, unparsable, or out of range.
func Int(key string, fallback, min, max int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < min || n > max {
		return fallback
	}
	return n
}
