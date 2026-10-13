package standardsync

import (
	"errors"
	"testing"
)

func bindableResolver(t *testing.T) *Resolver {
	t.Helper()
	cards := []StandardMetadataCard{
		mustCard(t, "did:integin:standard:asme-b30-5-2021", "ASME", "B30.5", 2021, LifecycleActive, "2021-05-01", ""),
		mustCard(t, "did:integin:standard:iso-4309-2010", "ISO", "4309", 2010, LifecycleActive, "2010-02-15", "did:integin:standard:iso-4309-1990"),
		mustCard(t, "did:integin:standard:iso-4309-1990", "ISO", "4309", 1990, LifecycleSuperseded, "1990-12-01", "did:integin:standard:iso-4309-2010"),
	}
	resolver, err := NewResolver(cards)
	if err != nil {
		t.Fatalf("NewResolver: %v", err)
	}
	return resolver
}

func TestBindToPackageResolvesActiveOnly(t *testing.T) {
	r := bindableResolver(t)
	didList := []string{
		"did:integin:standard:asme-b30-5-2021",
		"did:integin:standard:iso-4309-2010",
		"did:integin:standard:asme-b30-5-2021", // duplicate collapses
	}
	binding, err := BindToPackage(r, "pkg-hash-abc", "lifting-maintenance", didList)
	if err != nil {
		t.Fatalf("BindToPackage: %v", err)
	}
	if len(binding.StandardDIDs) != 2 {
		t.Fatalf("dids = %v, want 2 unique", binding.StandardDIDs)
	}
	if binding.BindingDigest != bindingsDigest("pkg-hash-abc", "lifting-maintenance", binding.StandardDIDs) {
		t.Fatal("digest does not match identity fields")
	}
	if !Verify(binding, r) {
		t.Fatal("Verify rejected a consistent binding")
	}

	tampered := binding
	tampered.PackageHash = "pkg-hash-tampered"
	if Verify(tampered, r) {
		t.Fatal("Verify accepted a tampered package hash")
	}
}

func TestBindToPackageRejectsSuperseded(t *testing.T) {
	r := bindableResolver(t)
	_, err := BindToPackage(r, "pkg-hash-abc", "lifting-maintenance", []string{"did:integin:standard:iso-4309-1990"})
	if !errors.Is(err, ErrNonBindingStandard) {
		t.Fatalf("err = %v, want ErrNonBindingStandard", err)
	}
}

func TestBindToPackageRejectsUnknown(t *testing.T) {
	r := bindableResolver(t)
	_, err := BindToPackage(r, "pkg-hash-abc", "lifting-maintenance", []string{"did:integin:standard:iso-99999-2021"})
	if !errors.Is(err, ErrUnresolvedStandard) {
		t.Fatalf("err = %v, want ErrUnresolvedStandard", err)
	}
}

func TestBindToPackageRequiresPackageHash(t *testing.T) {
	r := bindableResolver(t)
	if _, err := BindToPackage(r, "", "lifting-maintenance", []string{"did:integin:standard:asme-b30-5-2021"}); err == nil {
		t.Fatal("expected error for empty package hash")
	}
}

func TestManifestBindingJSONHasNoCanadaLeak(t *testing.T) {
	r := bindableResolver(t)
	binding, err := BindToPackage(r, "pkg-hash-abc", "lifting-maintenance", []string{"did:integin:standard:asme-b30-5-2021"})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := binding.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"ScopeAbstract", "scope_abstract"} {
		if contains(string(raw), forbidden) {
			t.Fatalf("binding JSON leaked %s", forbidden)
		}
	}
}
