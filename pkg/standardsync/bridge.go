package standardsync

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
)

// ErrUnresolvedStandard reports a DID that the resolver cannot evidence.
var ErrUnresolvedStandard = errors.New("standard DID could not be resolved")

// ErrNonBindingStandard reports a resolved card whose lifecycle state cannot
// be injected into a work order package (anything other than ACTIVE).
var ErrNonBindingStandard = errors.New("standard is not ACTIVE and cannot bind to a manifest")

// ManifestBindingSet attaches verified Standard DIDs to a work order package
// without mutating the signed manifest grammar. The digest covers the package
// hash, the assignment template, and the sorted verified DID list, so a client
// can prove binding integrity against an already-issued manifest signature.
type ManifestBindingSet struct {
	PackageHash   string                 `json:"package_hash"`
	TemplateCode  string                 `json:"template_code"`
	StandardDIDs  []string               `json:"standard_dids"`
	Resolved      []StandardMetadataCard `json:"resolved"`
	BindingDigest string                 `json:"binding_digest"`
	CreatedAt     string                 `json:"created_at_utc"`
}

// BindToPackage resolves each DID against the resolver, requires every card to
// be the ACTIVE binding revision of its series, and produces a deterministic
// digest over the package identity plus the sorted DID list. Duplicate DIDs
// are collapsed. The result is transport-agnostic so a route can attach it to
// a manifest response without altering packagemanifest's signed canon.
func BindToPackage(resolver *Resolver, packageHash, templateCode string, dids []string) (ManifestBindingSet, error) {
	if resolver == nil {
		return ManifestBindingSet{}, errors.New("standards resolver is required")
	}
	if strings.TrimSpace(packageHash) == "" {
		return ManifestBindingSet{}, errors.New("package hash is required")
	}
	seen := make(map[string]bool, len(dids))
	ordered := make([]string, 0, len(dids))
	for _, did := range strings.Split(strings.Join(dids, " "), " ") {
		did = strings.TrimSpace(did)
		if did == "" || seen[did] {
			continue
		}
		card, ok := resolver.CardByDID(did)
		if !ok {
			return ManifestBindingSet{}, fmt.Errorf("%w: %s", ErrUnresolvedStandard, did)
		}
		if card.LifecycleState != LifecycleActive || !resolver.VerifyBool(did) {
			return ManifestBindingSet{}, fmt.Errorf("%w: %s", ErrNonBindingStandard, did)
		}
		seen[did] = true
		ordered = append(ordered, did)
	}
	sort.Strings(ordered)

	resolved := make([]StandardMetadataCard, 0, len(ordered))
	for _, did := range ordered {
		card, _ := resolver.CardByDID(did)
		resolved = append(resolved, card)
	}

	binding := ManifestBindingSet{
		PackageHash:   packageHash,
		TemplateCode:  templateCode,
		StandardDIDs:  ordered,
		Resolved:      resolved,
		BindingDigest: bindingsDigest(packageHash, templateCode, ordered),
	}
	return binding, nil
}

// Verify checks that the binding's digest still matches its identity fields
// and that every listed DID remains ACTIVE in the resolver. It costs one
// digest recompute and one resolver lookup per DID.
func Verify(binding ManifestBindingSet, resolver *Resolver) bool {
	if binding.BindingDigest != bindingsDigest(binding.PackageHash, binding.TemplateCode, binding.StandardDIDs) {
		return false
	}
	if len(binding.StandardDIDs) == 0 {
		return true
	}
	for _, did := range binding.StandardDIDs {
		if !resolver.VerifyBool(did) {
			return false
		}
	}
	return true
}

// bindingsDigest is a deterministic SHA-256 over the package identity and the
// sorted DID list (already sorted by callers of BindToPackage).
func bindingsDigest(packageHash, templateCode string, dids []string) string {
	canonical, _ := json.Marshal(struct {
		PackageHash  string   `json:"package_hash"`
		TemplateCode string   `json:"template_code"`
		Standards    []string `json:"standards"`
	}{packageHash, templateCode, dids})
	sum := sha256.Sum256(canonical)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// MarshalJSON serializes the binding with citation-only card fields. Scope
// abstracts, asset lists, and tags are deliberately excluded: a manifest
// binding carries the verified identity of the standard, never payload prose.
func (b ManifestBindingSet) MarshalJSON() ([]byte, error) {
	type citedStandard struct {
		DID      string `json:"did"`
		Body     string `json:"body"`
		Code     string `json:"code"`
		Revision int    `json:"revision_year"`
		Title    string `json:"title"`
		State    string `json:"lifecycle_state"`
		StoreURL string `json:"official_store_url"`
	}
	resolved := make([]citedStandard, 0, len(b.Resolved))
	for _, card := range b.Resolved {
		resolved = append(resolved, citedStandard{
			DID:      card.StandardDID,
			Body:     card.StandardBody,
			Code:     card.Code,
			Revision: card.RevisionYear,
			Title:    card.Title,
			State:    string(card.LifecycleState),
			StoreURL: card.OfficialStoreURL,
		})
	}
	payload := struct {
		PackageHash   string          `json:"package_hash"`
		TemplateCode  string          `json:"template_code"`
		StandardDIDs  []string        `json:"standard_dids"`
		Resolved      []citedStandard `json:"resolved"`
		BindingDigest string          `json:"binding_digest"`
		CreatedAt     string          `json:"created_at_utc"`
	}{
		PackageHash:   b.PackageHash,
		TemplateCode:  b.TemplateCode,
		StandardDIDs:  b.StandardDIDs,
		Resolved:      resolved,
		BindingDigest: b.BindingDigest,
		CreatedAt:     b.CreatedAt,
	}
	return json.Marshal(payload)
}
