package server

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"integin/internal/manifestreceiptbridge"
	"integin/internal/packagemanifest"
	"integin/internal/packagemanifestapi"
	"integin/internal/workpackagepg"
)

const receiptAttachmentRunID = "0123456789abcdef0123456789abcdef"

func TestAttachPilotManifestReceiptBridgeLeavesAllAbsentContextInert(t *testing.T) {
	directory := t.TempDir()
	clearPilotReceiptContext(t)
	handler := receiptAttachmentHandler()
	if err := attachPilotManifestReceiptBridge(handler); err != nil {
		t.Fatalf("all-absent context returned error: %v", err)
	}
	if handler.Observer != nil || handler.ReplayCounter != nil {
		t.Fatal("all-absent context attached receipt components")
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("all-absent context created artifacts: %#v", entries)
	}
}

func TestAttachPilotManifestReceiptBridgeRejectsPartialOrUnsafeContext(t *testing.T) {
	for _, testCase := range []struct {
		name      string
		runID     string
		directory string
		version   string
	}{
		{name: "run id only", runID: receiptAttachmentRunID},
		{name: "directory only", directory: t.TempDir()},
		{name: "version only", version: "2"},
		{name: "unsafe directory", runID: receiptAttachmentRunID, directory: writeReceiptAttachmentFile(t), version: "2"},
		{name: "unsupported version", runID: receiptAttachmentRunID, directory: t.TempDir(), version: "3"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			setPilotReceiptContext(t, testCase.runID, testCase.directory, testCase.version)
			handler := receiptAttachmentHandler()
			if err := attachPilotManifestReceiptBridge(handler); err == nil {
				t.Fatal("expected context rejection")
			}
			if handler.Observer != nil || handler.ReplayCounter != nil {
				t.Fatal("rejected context attached receipt components")
			}
		})
	}
}

func TestAttachPilotManifestReceiptBridgeUsesExactDurableStoreForCounter(t *testing.T) {
	directory := t.TempDir()
	setPilotReceiptContext(t, receiptAttachmentRunID, directory, "2")
	handler := receiptAttachmentHandler()
	expectedStore, ok := handler.ReplayStore.(*workpackagepg.ManifestProofReplayStore)
	if !ok || expectedStore == nil {
		t.Fatalf("test handler replay store type = %T, want durable store", handler.ReplayStore)
	}
	if err := attachPilotManifestReceiptBridge(handler); err != nil {
		t.Fatalf("attach receipt bridge: %v", err)
	}
	if _, ok := handler.Observer.(*manifestreceiptbridge.HTTPObserver); !ok {
		t.Fatalf("observer type = %T, want HTTPObserver", handler.Observer)
	}
	counter, ok := handler.ReplayCounter.(*workpackagepg.ManifestProofReplayStore)
	if !ok || counter != expectedStore {
		t.Fatal("receipt counter is not the handler's exact durable replay store")
	}
}

func TestAttachPilotManifestReceiptBridgeRejectsNonDurableReplayStore(t *testing.T) {
	setPilotReceiptContext(t, receiptAttachmentRunID, t.TempDir(), "2")
	handler := &packagemanifestapi.Handler{ReplayStore: packagemanifest.NewInMemoryProofReplayStore(nil)}
	if err := attachPilotManifestReceiptBridge(handler); err == nil {
		t.Fatal("expected non-durable replay store rejection")
	}
	if handler.Observer != nil || handler.ReplayCounter != nil {
		t.Fatal("non-durable replay store attached receipt components")
	}
}

func TestAttachPilotManifestReceiptBridgeRejectsUnreadyDurableStore(t *testing.T) {
	setPilotReceiptContext(t, receiptAttachmentRunID, t.TempDir(), "2")
	handler := &packagemanifestapi.Handler{ReplayStore: &workpackagepg.ManifestProofReplayStore{}}
	if err := attachPilotManifestReceiptBridge(handler); err == nil {
		t.Fatal("expected unready durable replay store rejection")
	}
}

func TestPilotManifestHandlerDisabledByDefaultIgnoresReceiptContext(t *testing.T) {
	setPilotReceiptContext(t, receiptAttachmentRunID, t.TempDir(), "2")
	t.Setenv(pilotManifestRetrievalEnvironment, "")
	t.Setenv(pilotRuntimeEnvironment, "")
	t.Setenv(pilotManifestPrivateKeyEnvironment, "")
	t.Setenv(pilotManifestKeyIDEnvironment, "")
	handler, registry, err := PilotManifestHandlerFromEnvironment(nil, nil, nil, IsolatedPilotManifestAddress)
	if err != nil || handler != nil || registry != nil {
		t.Fatalf("default-disabled receipt context changed handler result: handler=%v registry=%v err=%v", handler, registry, err)
	}
}

func receiptAttachmentHandler() *packagemanifestapi.Handler {
	repository, err := workpackagepg.NewRepository(&sql.DB{})
	if err != nil {
		panic(err)
	}
	return &packagemanifestapi.Handler{ReplayStore: workpackagepg.NewManifestProofReplayStore(repository)}
}

func clearPilotReceiptContext(t *testing.T) {
	t.Helper()
	setPilotReceiptContext(t, "", "", "")
}

func setPilotReceiptContext(t *testing.T, runID, directory, version string) {
	t.Helper()
	t.Setenv(pilotManifestRunIDEnvironment, runID)
	t.Setenv(pilotManifestReceiptsEnvironment, directory)
	t.Setenv(pilotManifestReceiptVersionEnvironment, version)
}

func writeReceiptAttachmentFile(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(path, []byte("not a directory"), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}
