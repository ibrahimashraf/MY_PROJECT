package standards_vault

import (
	"context"
	"testing"

	"integin/internal/storage"
)

func TestStandardsVaultSearchIsTenantAwareAndDownloadsObjects(t *testing.T) {
	objects := storage.NewInMemoryStore()
	if err := objects.Put(context.Background(), storage.Object{Key: "standards/iso.pdf", ContentType: "application/pdf", Data: []byte("standard")}); err != nil {
		t.Fatal(err)
	}
	vault := New(objects)
	if err := vault.Add(Standard{ID: "standard-1", TenantID: "tenant-1", Code: "ISO 10855", Version: "2020", Title: "Offshore containers", FileKey: "standards/iso.pdf", Clauses: []Clause{{Number: "7.1", Title: "Inspection", Content: "periodic inspection requirements"}}}); err != nil {
		t.Fatal(err)
	}
	if err := vault.Add(Standard{ID: "standard-2", TenantID: "tenant-2", Code: "ISO OTHER", Version: "2020", Title: "Other", FileKey: "standards/iso.pdf"}); err != nil {
		t.Fatal(err)
	}
	results := vault.Search("tenant-1", "periodic")
	if len(results) != 1 || results[0].ID != "standard-1" {
		t.Fatalf("unexpected search: %#v", results)
	}
	if len(vault.Search("tenant-1", "ISO OTHER")) != 0 {
		t.Fatal("cross-tenant standard leaked")
	}
	object, err := vault.Download("tenant-1", "standard-1")
	if err != nil || string(object.Data) != "standard" {
		t.Fatalf("unexpected download: %#v %v", object, err)
	}
}
