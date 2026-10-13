package shortlinkpg

import (
	"os"
	"testing"
	"time"
)

func TestGetWebhookRetryIntervals_Defaults(t *testing.T) {
	os.Unsetenv("WEBHOOK_RETRY_INTERVALS")
	intervals := getWebhookRetryIntervals()
	if len(intervals) != 6 {
		t.Fatalf("expected 6 default intervals, got %d", len(intervals))
	}
	expected := []time.Duration{
		1 * time.Minute,
		5 * time.Minute,
		15 * time.Minute,
		1 * time.Hour,
		6 * time.Hour,
		24 * time.Hour,
	}
	for i, d := range expected {
		if intervals[i] != d {
			t.Fatalf("interval[%d] = %v, want %v", i, intervals[i], d)
		}
	}
}

func TestGetWebhookRetryIntervals_EmptyEnv(t *testing.T) {
	os.Setenv("WEBHOOK_RETRY_INTERVALS", "")
	defer os.Unsetenv("WEBHOOK_RETRY_INTERVALS")
	intervals := getWebhookRetryIntervals()
	if len(intervals) != 6 {
		t.Fatalf("expected 6 default intervals for empty env, got %d", len(intervals))
	}
}

func TestGetWebhookRetryIntervals_Custom(t *testing.T) {
	os.Setenv("WEBHOOK_RETRY_INTERVALS", "2m,10m,30m")
	defer os.Unsetenv("WEBHOOK_RETRY_INTERVALS")
	intervals := getWebhookRetryIntervals()
	if len(intervals) != 3 {
		t.Fatalf("expected 3 custom intervals, got %d", len(intervals))
	}
	if intervals[0] != 2*time.Minute {
		t.Fatalf("interval[0] = %v, want 2m", intervals[0])
	}
	if intervals[1] != 10*time.Minute {
		t.Fatalf("interval[1] = %v, want 10m", intervals[1])
	}
	if intervals[2] != 30*time.Minute {
		t.Fatalf("interval[2] = %v, want 30m", intervals[2])
	}
}

func TestGetWebhookRetryIntervals_InvalidFallsBackToDefaults(t *testing.T) {
	os.Setenv("WEBHOOK_RETRY_INTERVALS", "not-a-duration,also-bad")
	defer os.Unsetenv("WEBHOOK_RETRY_INTERVALS")
	intervals := getWebhookRetryIntervals()
	if len(intervals) != 6 {
		t.Fatalf("expected 6 default intervals for invalid env, got %d", len(intervals))
	}
}

func TestGetWebhookRetryIntervals_AllEmptyFallsBackToDefaults(t *testing.T) {
	os.Setenv("WEBHOOK_RETRY_INTERVALS", " , , ")
	defer os.Unsetenv("WEBHOOK_RETRY_INTERVALS")
	intervals := getWebhookRetryIntervals()
	if len(intervals) != 6 {
		t.Fatalf("expected 6 default intervals for whitespace-only env, got %d", len(intervals))
	}
}

func TestGetWebhookRetryIntervals_NegativeDurationFallsBackToDefaults(t *testing.T) {
	os.Setenv("WEBHOOK_RETRY_INTERVALS", "-1m,5m")
	defer os.Unsetenv("WEBHOOK_RETRY_INTERVALS")
	intervals := getWebhookRetryIntervals()
	if len(intervals) != 6 {
		t.Fatalf("expected 6 default intervals for negative duration, got %d", len(intervals))
	}
}
