// Package manifestreceiptbridge maps only closed source-redacted HTTP outcomes
// into v2 receipt attempts. It is not composed into any runtime by this file.
package manifestreceiptbridge

import (
	"context"

	"integin/internal/manifestreceipts"
	"integin/internal/packagemanifestapi"
)

// HTTPObserver is a non-authoritative receipt attempt sink. It writes no
// diagnostic receipt for observations outside the closed public case matrix.
type HTTPObserver struct {
	writer *manifestreceipts.V2Writer
}

// NewHTTPObserver returns a source-only mapper. Providing a nil writer keeps
// the mapper inert and is useful for defensive composition tests.
func NewHTTPObserver(writer *manifestreceipts.V2Writer) *HTTPObserver {
	return &HTTPObserver{writer: writer}
}

// Observe satisfies packagemanifestapi.ObservationSink. Receipt publication
// cannot alter the already-determined manifest handler HTTP response.
func (o *HTTPObserver) Observe(_ context.Context, observation packagemanifestapi.Observation) {
	if o == nil || o.writer == nil {
		return
	}
	receipt, ok := mapHTTPObservation(observation)
	if !ok {
		return
	}
	_ = o.writer.EmitWithFailure(receipt)
}

func mapHTTPObservation(observation packagemanifestapi.Observation) (manifestreceipts.V2Observation, bool) {
	caseValue, outcome, ok := mapHTTPCase(observation)
	if !ok {
		return manifestreceipts.V2Observation{}, false
	}
	status := observation.HTTPStatus
	receipt := manifestreceipts.V2Observation{
		Case:            caseValue,
		EventSource:     manifestreceipts.V2EventSourceManifestHTTP,
		ObservedOutcome: outcome,
		Transport:       manifestreceipts.V2Transport{Kind: manifestreceipts.V2TransportHTTP, HTTPStatus: &status},
		ReplayState:     manifestreceipts.V2ReplayState{Scope: manifestreceipts.V2ReplayScopeNotApplicable},
		ManifestID:      observation.ManifestID,
	}
	if caseValue != manifestreceipts.CaseValidProof {
		return receipt, true
	}
	if !observation.ReplayAccepted || observation.ReplayBefore == nil || observation.ReplayAfter == nil {
		return receipt, true
	}
	delta := *observation.ReplayAfter - *observation.ReplayBefore
	receipt.ReplayState = manifestreceipts.V2ReplayState{
		Scope:  manifestreceipts.V2ReplayScopeVerifiedTenant,
		Before: observation.ReplayBefore,
		After:  observation.ReplayAfter,
		Delta:  &delta,
	}
	return receipt, true
}

func mapHTTPCase(observation packagemanifestapi.Observation) (manifestreceipts.Case, string, bool) {
	switch {
	case observation.Outcome == "manifest_issued" && observation.ReasonCode == "valid_proof" && observation.HTTPStatus == 200:
		return manifestreceipts.CaseValidProof, "proof_valid", true
	case observation.Outcome == "replay_rejected" && observation.ReasonCode == "replay" && observation.HTTPStatus == 409:
		return manifestreceipts.CaseReplay, "replay_rejected", true
	case observation.Outcome == "proof_rejected" && observation.ReasonCode == "signature_invalid" && observation.HTTPStatus == 401:
		return manifestreceipts.CaseSignatureInvalid, "signature_invalid", true
	case observation.Outcome == "proof_rejected" && observation.ReasonCode == "package_hash_invalid" && observation.HTTPStatus == 403:
		return manifestreceipts.CasePackageHashInvalid, "package_hash_invalid", true
	case observation.Outcome == "proof_rejected" && observation.ReasonCode == "expired" && (observation.HTTPStatus == 401 || observation.HTTPStatus == 403):
		return manifestreceipts.CaseExpired, "expired", true
	case observation.Outcome == "proof_rejected" && observation.ReasonCode == "authority_mismatch" && (observation.HTTPStatus == 401 || observation.HTTPStatus == 403):
		return manifestreceipts.CaseAuthorityMismatch, "authority_mismatch", true
	case observation.Outcome == "proof_rejected" && observation.ReasonCode == "key_unknown" && observation.HTTPStatus == 401:
		return manifestreceipts.CaseKeyUnknown, "key_unknown", true
	default:
		return "", "", false
	}
}
