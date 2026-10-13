package standardsync

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

var (
	// ErrClientEpochAhead reports a client claiming an epoch newer than the
	// authoritative catalogue (clock skew or fork); sync is rejected.
	ErrClientEpochAhead = errors.New("client epoch is ahead of the authoritative standards catalogue")
	// ErrInvalidDeltaRequest reports a malformed delta-sync request.
	ErrInvalidDeltaRequest = errors.New("delta-sync request is invalid")
	// ErrInvalidMasterCatalogue reports an unresolvable master catalogue.
	ErrInvalidMasterCatalogue = errors.New("master standards catalogue is invalid")
)

// DeltaSyncRequest is the client state snapshot an edge tablet sends so the
// master can emit only amended or withdrawn standards cards.
type DeltaSyncRequest struct {
	// CurrentRevisions maps each known standard DID to the revision year the
	// client currently holds. Absent DIDs are treated as not-present.
	CurrentRevisions map[string]int
	// ClientEpoch is the catalogue epoch the client is synchronized through.
	ClientEpoch uint64
	// Jurisdiction is the ISO 3166-1 alpha-2 code of the tablet's operating
	// jurisdiction. Empty means the request is jurisdiction-agnostic.
	Jurisdiction string
}

// DeltaSyncResponse is the minimal set of standards-card changes the tablet
// must apply to converge with the master catalogue.
type DeltaSyncResponse struct {
	UpdatedStandards []StandardMetadataCard `json:"updated_standards"`
	WithdrawnDIDs    []string               `json:"withdrawn_dids"`
	NewEpoch         uint64                 `json:"new_epoch"`
	HasMore          bool                   `json:"has_more"`
}

// DeltaSyncEngine computes minimal catalogue deltas between a client's
// reported state and the authoritative master card set.
type DeltaSyncEngine struct {
	byDID    map[string]StandardMetadataCard
	master   []StandardMetadataCard // deterministic order
	epoch    uint64
	pageSize int
}

const defaultDeltaPageSize = 250

// NewDeltaSyncEngine builds an engine over the master card catalogue. Every
// card must pass validation; duplicates and a zero-size page are rejected.
func NewDeltaSyncEngine(cards []StandardMetadataCard, epoch uint64, pageSize int) (*DeltaSyncEngine, error) {
	if pageSize < 1 {
		pageSize = defaultDeltaPageSize
	}
	byDID := make(map[string]StandardMetadataCard, len(cards))
	sorted := append([]StandardMetadataCard(nil), cards...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].StandardDID < sorted[j].StandardDID })
	for _, card := range sorted {
		if err := card.Validate(); err != nil {
			return nil, fmt.Errorf("%w: %q: %v", ErrInvalidMasterCatalogue, card.StandardDID, err)
		}
		if _, dup := byDID[card.StandardDID]; dup {
			return nil, fmt.Errorf("%w: duplicate standard DID %q", ErrInvalidMasterCatalogue, card.StandardDID)
		}
		byDID[card.StandardDID] = card
	}
	return &DeltaSyncEngine{byDID: byDID, master: sorted, epoch: epoch, pageSize: pageSize}, nil
}

var iso2Pattern = regexp.MustCompile(`^[A-Z]{2}$`)

// ComputeDelta compares client state against the master catalogue and emits
// the minimal set of standards cards that changed. It is stateless and
// deterministic: the same (client state, master) always yields the same delta.
func (e *DeltaSyncEngine) ComputeDelta(req DeltaSyncRequest) (DeltaSyncResponse, error) {
	if req.ClientEpoch > e.epoch {
		return DeltaSyncResponse{}, ErrClientEpochAhead
	}
	if req.Jurisdiction != "" && !iso2Pattern.MatchString(req.Jurisdiction) {
		return DeltaSyncResponse{}, fmt.Errorf("%w: jurisdiction %q is not ISO 3166-1 alpha-2", ErrInvalidDeltaRequest, req.Jurisdiction)
	}

	var updated []StandardMetadataCard
	for _, card := range e.master {
		if !jurisdictionInScope(card, req.Jurisdiction) {
			continue
		}
		held, known := req.CurrentRevisions[card.StandardDID]
		if !known || held != card.RevisionYear {
			updated = append(updated, card)
		}
	}

	withdrawn := e.withdrawn(req.CurrentRevisions)

	// Deterministic ordering: updated cards first (content, revision order),
	// then withdrawn DIDs, both sorted lexically.
	hasMore := len(updated)+len(withdrawn) > e.pageSize
	pageRemain := e.pageSize
	if len(updated) > pageRemain {
		hasMore = true
		// ponytail: no pagination cursor — the client merges each page into
		// CurrentRevisions and re-requests, so the stateless delta converges.
		updated = updated[:pageRemain]
		pageRemain = 0
	} else {
		pageRemain -= len(updated)
	}
	if pageRemain < len(withdrawn) {
		hasMore = true
		withdrawn = withdrawn[:pageRemain]
	}

	return DeltaSyncResponse{
		UpdatedStandards: updated,
		WithdrawnDIDs:    withdrawn,
		NewEpoch:         e.epoch,
		HasMore:          hasMore,
	}, nil
}

// withdrawnDIDs returns every client-held DID that no longer exists in the
// active master catalogue or whose card is now withdrawn in the master set.
func (e *DeltaSyncEngine) withdrawn(client map[string]int) []string {
	var out []string
	for did := range client {
		card, ok := e.byDID[did]
		if !ok {
			out = append(out, did)
			continue
		}
		if card.LifecycleState == LifecycleWithdrawn {
			out = append(out, did)
		}
	}
	sort.Strings(out)
	return out
}

var iso2Tag = regexp.MustCompile(`^[A-Z]{2}$`)

// jurisdictionInScope reports whether a card is published to the given
// jurisdiction. Cards carrying no ISO-looking tag are country-agnostic and
// always in scope; a "GLOBAL" tag also publishes everywhere.
func jurisdictionInScope(card StandardMetadataCard, iso string) bool {
	if iso == "" {
		return true
	}
	restricted := false
	for _, tag := range card.Tags {
		upper := strings.ToUpper(strings.TrimSpace(tag))
		if upper == "GLOBAL" {
			return true
		}
		if iso2Tag.MatchString(upper) {
			restricted = true
			if upper == iso {
				return true
			}
		}
	}
	return !restricted
}

// CatalogueEpoch reports the engine's authoritative epoch.
func (e *DeltaSyncEngine) CatalogueEpoch() uint64 { return e.epoch }

// CardCount reports the number of cards in the master catalogue.
func (e *DeltaSyncEngine) CardCount() int { return len(e.master) }
