package timecal

import (
	"testing"
)

func TestJulianDayGregorianRoundTrip(t *testing.T) {
	// Known JDN: 2000-01-01 = JDN 2451545
	jdn := JulianDayNumber(2000, 1, 1)
	if jdn != 2451545 {
		t.Fatalf("JDN for 2000-01-01 expected 2451545, got %d", jdn)
	}
	y, m, d := JDNToGregorian(jdn)
	if y != 2000 || m != 1 || d != 1 {
		t.Fatalf("JDN roundtrip failed: got %d-%d-%d", y, m, d)
	}
}

func TestHijriCalendarConversion(t *testing.T) {
	// 2024-09-12 Gregorian -> approximate Hijri 1446 AH
	hY, hM, _ := HijriFromGregorian(2024, 9, 12)
	if hY != 1446 {
		t.Fatalf("Hijri year mismatch: expected 1446, got %d", hY)
	}
	if hM < 1 || hM > 12 {
		t.Fatalf("Hijri month out of range: %d", hM)
	}
}
