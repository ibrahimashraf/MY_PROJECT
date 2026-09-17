package env

import (
	"testing"
	"time"
)

func TestSecondsFallback(t *testing.T) {
	if got := Seconds("INTEGIN_TEST_UNSET_XYZ", 5*time.Second, 1, 60); got != 5*time.Second {
		t.Fatalf("unset: got %v", got)
	}
	t.Setenv("INTEGIN_TEST_SECONDS", "10")
	if got := Seconds("INTEGIN_TEST_SECONDS", 5*time.Second, 1, 60); got != 10*time.Second {
		t.Fatalf("set: got %v", got)
	}
	t.Setenv("INTEGIN_TEST_SECONDS", "9999")
	if got := Seconds("INTEGIN_TEST_SECONDS", 5*time.Second, 1, 60); got != 5*time.Second {
		t.Fatalf("clamp: got %v", got)
	}
	t.Setenv("INTEGIN_TEST_SECONDS", "bogus")
	if got := Seconds("INTEGIN_TEST_SECONDS", 5*time.Second, 1, 60); got != 5*time.Second {
		t.Fatalf("parse: got %v", got)
	}
}

func TestIntClamp(t *testing.T) {
	t.Setenv("INTEGIN_TEST_INT", "3")
	if got := Int("INTEGIN_TEST_INT", 5, 1, 25); got != 3 {
		t.Fatalf("set: got %d", got)
	}
	t.Setenv("INTEGIN_TEST_INT", "999")
	if got := Int("INTEGIN_TEST_INT", 5, 1, 25); got != 5 {
		t.Fatalf("clamp: got %d", got)
	}
}
