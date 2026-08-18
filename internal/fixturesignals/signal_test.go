package fixturesignals

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestEmitWritesClosedRedactedSignal(t *testing.T) {
	directory := t.TempDir()
	now := time.Date(2026, time.August, 18, 18, 0, 0, 0, time.UTC)
	emitted, err := Emit(directory, SignalContractVersion, func() time.Time { return now }, bytes.NewReader(bytes.Repeat([]byte{0x42}, 16)))
	if err != nil || !emitted {
		t.Fatalf("emit result = emitted:%t err:%v", emitted, err)
	}
	content, err := os.ReadFile(filepath.Join(directory, SignalFileName))
	if err != nil {
		t.Fatalf("read signal: %v", err)
	}
	var signal map[string]any
	if err := json.Unmarshal(content, &signal); err != nil {
		t.Fatalf("decode signal: %v", err)
	}
	if len(signal) != 4 || signal["contract_version"] != SignalContractVersion || signal["step"] != SignalStepGenerated || signal["signal_id"] != "42424242424242424242424242424242" || signal["generated_at"] != "2026-08-18T18:00:00Z" {
		t.Fatalf("unexpected fixture signal: %#v", signal)
	}
}

func TestEmitAllowsAbsentContextButRejectsPartialInvalidAndDuplicateContext(t *testing.T) {
	emitted, err := Emit("", "", time.Now, bytes.NewReader(nil))
	if err != nil || emitted {
		t.Fatalf("absent context result = emitted:%t err:%v", emitted, err)
	}
	if _, err := Emit(t.TempDir(), "", time.Now, bytes.NewReader(bytes.Repeat([]byte{1}, 16))); err == nil {
		t.Fatal("partial context was accepted")
	}
	if _, err := Emit(t.TempDir(), "2", time.Now, bytes.NewReader(bytes.Repeat([]byte{1}, 16))); err == nil {
		t.Fatal("unsupported version was accepted")
	}
	directory := t.TempDir()
	if _, err := Emit(directory, SignalContractVersion, time.Now, bytes.NewReader(bytes.Repeat([]byte{1}, 16))); err != nil {
		t.Fatalf("first emit: %v", err)
	}
	if _, err := Emit(directory, SignalContractVersion, time.Now, bytes.NewReader(bytes.Repeat([]byte{2}, 16))); !errors.Is(err, ErrDuplicateSignal) {
		t.Fatalf("duplicate error = %v", err)
	}
}
