package id

import (
	"testing"
	"time"
)

func TestNewV7IsValid(t *testing.T) {
	u, err := NewV7()
	if err != nil {
		t.Fatalf("NewV7 error: %v", err)
	}
	if len(u) != 36 {
		t.Fatalf("uuid length = %d, want 36", len(u))
	}
	if !IsValidV7(u) {
		t.Fatalf("generated uuid %q failed IsValidV7", u)
	}
}

func TestNewV7MonotonicOrdering(t *testing.T) {
	const n = 10000 // exceeds the 4096-per-millisecond counter ceiling
	prev := ""
	for i := 0; i < n; i++ {
		u, err := NewV7()
		if err != nil {
			t.Fatalf("NewV7 at %d: %v", i, err)
		}
		if u <= prev {
			t.Fatalf("uuid not strictly increasing at %d: %q <= %q", i, u, prev)
		}
		if !IsValidV7(u) {
			t.Fatalf("uuid %q not valid at iteration %d", u, i)
		}
		prev = u
	}
}

func TestNewV7FromTimeRoundTrip(t *testing.T) {
	times := []time.Time{
		time.UnixMilli(0).UTC(),
		time.UnixMilli(1700000000123).UTC(),
		time.Date(2024, 6, 1, 12, 30, 45, 0, time.UTC),
		time.Now().UTC().Truncate(time.Millisecond),
	}
	for _, tc := range times {
		u, err := NewV7FromTime(tc)
		if err != nil {
			t.Fatalf("NewV7FromTime(%v): %v", tc, err)
		}
		got, err := ParseTime(u)
		if err != nil {
			t.Fatalf("ParseTime(%q): %v", u, err)
		}
		if !got.Equal(tc) {
			t.Fatalf("round trip mismatch: got %v, want %v (uuid %q)", got, tc, u)
		}
	}
}

func TestParseTimeIdenticalTimestampOrdering(t *testing.T) {
	u1, err := NewV7FromTime(time.UnixMilli(1700000000000).UTC())
	if err != nil {
		t.Fatalf("NewV7FromTime: %v", err)
	}
	u2, err := NewV7FromTime(time.UnixMilli(1700000000000).UTC())
	if err != nil {
		t.Fatalf("NewV7FromTime: %v", err)
	}
	if u1 >= u2 {
		t.Fatalf("expected monotonic ordering across identical timestamps: %q >= %q", u1, u2)
	}
	t1, err := ParseTime(u1)
	if err != nil {
		t.Fatalf("ParseTime(%q): %v", u1, err)
	}
	t2, err := ParseTime(u2)
	if err != nil {
		t.Fatalf("ParseTime(%q): %v", u2, err)
	}
	if !t1.Equal(t2) {
		t.Fatalf("timestamps differ for same-ms uuid pair: %v vs %v", t1, t2)
	}
}

func TestIsValidV7RejectsNonV7(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{"canonical v7 zero timestamp", "00000000-0000-7000-8000-000000000000", true},
		{"v4 uuid", "550e8400-e29b-41d4-a716-446655440000", false},
		{"version nibble zero", "00000000-0000-0000-8000-000000000000", false},
		{"variant bits 11", "00000000-0000-7000-c000-000000000000", false},
		{"wrong length", "00000000-0000-7000-8000-00000000000", false},
		{"missing hyphen", "00000000X0000-7000-8000-000000000000", false},
		{"non hex digit", "zz000000-0000-7000-8000-000000000000", false},
		{"uppercase not accepted as valid parse", "00000000-0000-7000-8000-0000000000ZZ", false},
		{"empty string", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidV7(tt.in); got != tt.want {
				t.Fatalf("IsValidV7(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestNewV7BoundedAllocations(t *testing.T) {
	allocs := testing.AllocsPerRun(100, func() {
		u, err := NewV7()
		if err != nil {
			t.Fatalf("NewV7: %v", err)
		}
		_ = u
	})
	// The single mandatory allocation is the returned string itself.
	if allocs > 1.0 {
		t.Fatalf("NewV7 allocated %.2f objects per run; want <= 1.0 (the returned string)", allocs)
	}
}

func BenchmarkNewV7(b *testing.B) {
	b.ReportAllocs()
	var s string
	var err error
	for i := 0; i < b.N; i++ {
		s, err = NewV7()
		if err != nil {
			b.Fatalf("NewV7: %v", err)
		}
	}
	_ = s
}
