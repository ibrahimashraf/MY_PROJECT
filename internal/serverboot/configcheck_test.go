package serverboot

import (
	"log"
	"os"
	"strings"
	"testing"
)

func captureLogOutput(fn func()) string {
	r, w, err := os.Pipe()
	if err != nil {
		panic(err)
	}
	log.SetOutput(w)
	log.SetFlags(0)
	fn()
	w.Close()
	out := make([]byte, 4096)
	n, _ := r.Read(out)
	log.SetOutput(os.Stderr)
	log.SetFlags(log.LstdFlags)
	return string(out[:n])
}

func TestWarnIfUnconfigured_TSANotProvisioned(t *testing.T) {
	out := captureLogOutput(func() {
		warnIfUnconfigured(false, true, true, true)
	})
	if !strings.Contains(out, "TSA roots not provisioned") {
		t.Fatalf("expected TSA warning, got %q", out)
	}
	if strings.Contains(out, "TUS") {
		t.Fatalf("unexpected TUS warning: %q", out)
	}
}

func TestWarnIfUnconfigured_TUSDisabled(t *testing.T) {
	out := captureLogOutput(func() {
		warnIfUnconfigured(true, false, true, true)
	})
	if !strings.Contains(out, "TUS") || !strings.Contains(out, "disabled") {
		t.Fatalf("expected TUS disabled warning, got %q", out)
	}
	if strings.Contains(out, "TSA") {
		t.Fatalf("unexpected TSA warning: %q", out)
	}
}

func TestWarnIfUnconfigured_TUSWithoutOIDC(t *testing.T) {
	out := captureLogOutput(func() {
		warnIfUnconfigured(true, true, false, true)
	})
	if !strings.Contains(out, "OIDC validator") {
		t.Fatalf("expected OIDC warning, got %q", out)
	}
	if strings.Contains(out, "TSA") {
		t.Fatalf("unexpected TSA warning: %q", out)
	}
}

func TestWarnIfUnconfigured_AllConfiguredSilent(t *testing.T) {
	out := captureLogOutput(func() {
		warnIfUnconfigured(true, true, true, true)
	})
	if out != "" {
		t.Fatalf("expected no warnings when fully configured, got %q", out)
	}
}

func TestWarnIfUnconfigured_TSAAndTUSBothUnconfigured(t *testing.T) {
	out := captureLogOutput(func() {
		warnIfUnconfigured(false, false, false, true)
	})
	if !strings.Contains(out, "TSA roots not provisioned") {
		t.Fatalf("expected TSA warning, got %q", out)
	}
	if !strings.Contains(out, "TUS") {
		t.Fatalf("expected TUS warning, got %q", out)
	}
}

func TestWarnIfUnconfigured_EvidenceUnmounted(t *testing.T) {
	out := captureLogOutput(func() {
		warnIfUnconfigured(true, true, true, false)
	})
	if !strings.Contains(out, "/evidence") || !strings.Contains(out, "INTEGIN_EVIDENCE_STORE") {
		t.Fatalf("expected /evidence warning, got %q", out)
	}
	if strings.Contains(out, "TSA") || strings.Contains(out, "TUS") {
		t.Fatalf("unexpected TSA/TUS warning: %q", out)
	}
}
