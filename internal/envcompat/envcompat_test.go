package envcompat

import (
	"os"
	"testing"
)

func TestMirrorCopiesNewToUnsetOld(t *testing.T) {
	t.Setenv("INTEGIN_PROBE_NEW_VAR", "")
	os.Unsetenv("INTEGIN_PROBE_NEW_VAR")
	t.Setenv("INTEGIN_PROBE_NEW_VAR", "from-new")
	Mirror()
	if got := os.Getenv("INTEGIN_PROBE_NEW_VAR"); got != "from-new" {
		t.Fatalf("INTEGIN_PROBE_NEW_VAR = %q, want %q", got, "from-new")
	}
}

func TestMirrorOldWinsWhenBothSet(t *testing.T) {
	t.Setenv("INTEGIN_PROBE_BOTH_VAR", "old-kept")
	t.Setenv("INTEGIN_PROBE_BOTH_VAR", "new-ignored")
	Mirror()
	if got := os.Getenv("INTEGIN_PROBE_BOTH_VAR"); got != "old-kept" {
		t.Fatalf("INTEGIN_PROBE_BOTH_VAR = %q, want %q", got, "old-kept")
	}
}

func TestMirrorIgnoresUnrelated(t *testing.T) {
	t.Setenv("SOME_OTHER_VAR", "x")
	Mirror()
	if _, exists := os.LookupEnv("INTEGIN_SOME_OTHER_VAR"); exists {
		t.Fatal("Mirror created INTEGIN_SOME_OTHER_VAR from unrelated input")
	}
}
