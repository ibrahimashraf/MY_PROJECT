// Package edgecoordinator implements the Sovereign Edge Appliance daemon:
// exclusive sub-assembly leasing with overrun lifecycle, offline authority
// epoch ingestion from removable media, and in-chain Merkle override commits.
//
// All state transitions are tenant-scoped, fail-closed, and idempotent under
// concurrent sweeps. Tenant isolation is enforced twice: SQL predicates carry
// tenant_id, and every transaction sets the integin.tenant_id GUC so FORCE
// RLS policies hold for constrained roles (never rely on superuser bypass).
package edgecoordinator

import (
	"context"
	"crypto"
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"integin/pkg/domain"
)

const (
	EventTypeAdminOverride = "LEASE_OVERRUN_ADMIN_OVERRIDE"
	ReasonLeaseOverrun     = "LEASE_OVERRUN_EXPIRED"
)

var (
	ErrStaleBundleEpoch       = errors.New("epoch_ingest: bundle epoch is not strictly greater than current high-water")
	ErrInvalidBundleSignature = errors.New("epoch_ingest: root authority signature verification failed")
	ErrBundleExpired          = errors.New("epoch_ingest: epoch update bundle has expired")
	ErrBundleFutureDated      = errors.New("epoch_ingest: epoch update bundle issued in the future beyond allowable skew")
	ErrBundleInvalidWindow    = errors.New("epoch_ingest: bundle issued_at is after valid_until")
	ErrPathTraversal          = errors.New("epoch_ingest: bundle path escapes configured mount directory")
	ErrLeaseNotFound          = errors.New("coordinator: sub-assembly lease not found")
	ErrLeaseNotActive         = errors.New("coordinator: lease is not in ACTIVE state")
	ErrLeaseExpired           = errors.New("coordinator: lease valid_until expired prior to submission")
	ErrLeaseEpochMismatch     = errors.New("coordinator: record lease epoch does not match active tombstone epoch")
	ErrDraftNotFound          = errors.New("coordinator: referenced draft work order not found or already merged")
	ErrCorruptedMerkleChain   = errors.New("coordinator: previous merkle root has invalid byte length")
	ErrLeaseActiveConflict    = errors.New("coordinator: sub-assembly currently has an active or unmerged lease")
)

func binaryWrite(h hash.Hash, data any) error {
	return binary.Write(h, binary.BigEndian, data)
}

// beginTenantTx opens a transaction with the tenant GUC set (is_local), so
// FORCE RLS policies hold for constrained roles. Callers must Commit/Rollback.
func beginTenantTx(ctx context.Context, db *sql.DB, tenantID string) (*sql.Tx, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `SELECT set_config('integin.tenant_id', $1, true)`, tenantID); err != nil {
		_ = tx.Rollback()
		return nil, fmt.Errorf("failed to set tenant context: %w", err)
	}
	return tx, nil
}

type transitionRecord struct {
	subAssemblyID    string
	parentAssetDID   string
	assignedDeviceID string
	inspectorID      string
	leaseEpoch       uint64
}

// -------------------------------------------------------------------------
// 1. LEASE COORDINATOR & OVERRUN LIFECYCLE
// -------------------------------------------------------------------------

type LeaseCoordinator struct {
	db               *sql.DB
	tenantID         string
	orgID            string
	edgeAuthorityDID string // Used to stamp locally generated tombstones
	stopOnce         sync.Once
	stopChan         chan struct{}
	wg               sync.WaitGroup
	interval         time.Duration
	logger           *log.Logger
}

func NewLeaseCoordinator(db *sql.DB, tenantID, orgID, edgeAuthorityDID string, sweepInterval time.Duration, logger *log.Logger) *LeaseCoordinator {
	if sweepInterval <= 0 {
		sweepInterval = 10 * time.Second
	}
	if logger == nil {
		logger = log.New(os.Stderr, "[edgecoordinator] ", log.LstdFlags|log.LUTC)
	}
	return &LeaseCoordinator{
		db:               db,
		tenantID:         tenantID,
		orgID:            orgID,
		edgeAuthorityDID: edgeAuthorityDID,
		stopChan:         make(chan struct{}),
		interval:         sweepInterval,
		logger:           logger,
	}
}

func (c *LeaseCoordinator) Start(ctx context.Context) {
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		ticker := time.NewTicker(c.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-c.stopChan:
				return
			case now := <-ticker.C:
				if err := c.SweepExpiredLeases(ctx, now); err != nil {
					c.logger.Printf("ERROR: lease sweep failed: %v", err)
				}
			}
		}
	}()
}

func (c *LeaseCoordinator) Stop() {
	c.stopOnce.Do(func() {
		close(c.stopChan)
	})
	c.wg.Wait()
}

// DispatchLease issues or renews an exclusive lease. Re-issuance is permitted
// ONLY when the stored lease is terminal (RELEASED or COMPLETED) and the
// incoming epoch is strictly greater. Anything else fails closed with
// ErrLeaseActiveConflict (retryable after the blocking lease resolves).
func (c *LeaseCoordinator) DispatchLease(ctx context.Context, lease domain.SubAssemblyLease) error {
	const query = `
		INSERT INTO sub_assembly_leases (
			tenant_id, organization_id, sub_assembly_id, parent_asset_did,
			assigned_device_id, inspector_id, required_skill, criticality_class,
			lease_epoch, lease_start, valid_until, state, updated_at
		) VALUES (
			$1, $2, $3, $4,
			$5, $6, $7, $8,
			$9, $10, $11, 'ACTIVE', $10
		)
		ON CONFLICT (tenant_id, sub_assembly_id) DO UPDATE
		SET parent_asset_did = EXCLUDED.parent_asset_did,
		    assigned_device_id = EXCLUDED.assigned_device_id,
		    inspector_id = EXCLUDED.inspector_id,
		    required_skill = EXCLUDED.required_skill,
		    criticality_class = EXCLUDED.criticality_class,
		    lease_epoch = EXCLUDED.lease_epoch,
		    lease_start = EXCLUDED.lease_start,
		    valid_until = EXCLUDED.valid_until,
		    state = 'ACTIVE',
		    updated_at = EXCLUDED.updated_at
		WHERE sub_assembly_leases.state IN ('RELEASED', 'COMPLETED')
		  AND EXCLUDED.lease_epoch > sub_assembly_leases.lease_epoch;
	`
	tx, err := beginTenantTx(ctx, c.db, c.tenantID)
	if err != nil {
		return fmt.Errorf("failed to begin dispatch tx: %w", err)
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(ctx, query,
		c.tenantID, c.orgID, lease.SubAssemblyID, lease.ParentAssetDID,
		lease.AssignedDeviceID, lease.InspectorID, lease.RequiredSkill, string(lease.CriticalityClass),
		lease.LeaseEpoch, lease.LeaseStart, lease.ValidUntil,
	)
	if err != nil {
		return fmt.Errorf("failed to dispatch lease for %s: %w", lease.SubAssemblyID, err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check dispatch rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("%w: sub-assembly %s has active/unmerged lease or non-monotonic epoch", ErrLeaseActiveConflict, lease.SubAssemblyID)
	}

	return tx.Commit()
}

// SubmitInspection records a normal execution completion within the valid
// lease window.
//
// Concurrency & Retry Semantics:
//  1. If valid_until >= edgeNow, the lease atomically transitions to COMPLETED.
//  2. If valid_until has passed prior to commit, an inline JIT transition marks
//     the lease OVERRUN_DRAFT, provisions tombstone + draft artifacts, and
//     returns ErrLeaseExpired.
//  3. Spurious ACTIVE / interleaved re-dispatch: if a fresh ACTIVE lease lands
//     between the failed in-window UPDATE and the diagnostic SELECT, the method
//     fails closed with ErrLeaseNotActive naming state ACTIVE. Callers may retry
//     after re-fetching the updated lease manifest and epoch.
func (c *LeaseCoordinator) SubmitInspection(ctx context.Context, subAssemblyID string, edgeNow time.Time) error {
	tx, err := beginTenantTx(ctx, c.db, c.tenantID)
	if err != nil {
		return fmt.Errorf("failed to begin submit tx: %w", err)
	}
	defer tx.Rollback()

	// 1. Time-of-check must be the commit instant (valid_until >= edgeNow)
	const query = `
		UPDATE sub_assembly_leases
		SET state = 'COMPLETED',
		    updated_at = $3
		WHERE tenant_id = $1
		  AND sub_assembly_id = $2
		  AND state = 'ACTIVE'
		  AND valid_until >= $3;
	`
	res, err := tx.ExecContext(ctx, query, c.tenantID, subAssemblyID, edgeNow)
	if err != nil {
		return fmt.Errorf("failed to commit inspection completion: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 1 {
		return tx.Commit()
	}

	// 2. Zero rows affected: diagnose cause under row lock
	var currentState string
	var validUntil time.Time
	var parentAssetDID string
	var assignedDeviceID string
	var inspectorID string
	var leaseEpoch uint64

	const checkQuery = `
		SELECT state, valid_until, parent_asset_did, assigned_device_id, inspector_id, lease_epoch
		FROM sub_assembly_leases
		WHERE tenant_id = $1 AND sub_assembly_id = $2
		FOR UPDATE;
	`
	err = tx.QueryRowContext(ctx, checkQuery, c.tenantID, subAssemblyID).Scan(
		&currentState, &validUntil, &parentAssetDID, &assignedDeviceID, &inspectorID, &leaseEpoch,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrLeaseNotFound
		}
		return fmt.Errorf("failed to inspect lease state on zero update: %w", err)
	}

	// 3. JIT Overrun Transition: lease was ACTIVE but expired before the sweeper tick
	if currentState == "ACTIVE" {
		if edgeNow.After(validUntil) {
			const updateOverrun = `
				UPDATE sub_assembly_leases
				SET state = 'OVERRUN_DRAFT', updated_at = $3
				WHERE tenant_id = $1 AND sub_assembly_id = $2;
			`
			if _, err := tx.ExecContext(ctx, updateOverrun, c.tenantID, subAssemblyID, edgeNow); err != nil {
				return fmt.Errorf("failed to transition lease to overrun: %w", err)
			}

			t := transitionRecord{
				subAssemblyID:    subAssemblyID,
				parentAssetDID:   parentAssetDID,
				assignedDeviceID: assignedDeviceID,
				inspectorID:      inspectorID,
				leaseEpoch:       leaseEpoch,
			}
			if err := c.provisionOverrunArtifacts(ctx, tx, t, edgeNow); err != nil {
				return fmt.Errorf("failed to provision overrun artifacts on JIT overrun: %w", err)
			}

			if err := tx.Commit(); err != nil {
				return fmt.Errorf("failed to commit JIT overrun: %w", err)
			}
			return fmt.Errorf("%w: valid_until was %s, now is %s (transitioned to OVERRUN_DRAFT)", ErrLeaseExpired, validUntil, edgeNow)
		}

		// Reached if valid_until is in the future: a new ACTIVE lease was
		// committed between the failed in-window UPDATE and the diagnostic
		// SELECT FOR UPDATE (concurrent re-dispatch).
		return fmt.Errorf("%w: lease is in state ACTIVE (concurrent re-dispatch detected)", ErrLeaseNotActive)
	}

	return fmt.Errorf("%w: lease is in state %s", ErrLeaseNotActive, currentState)
}

// SweepExpiredLeases atomically transitions expired ACTIVE leases to
// OVERRUN_DRAFT, provisioning tombstone + draft artifacts per transition.
// Placeholders strictly match arguments ($1 = tenant_id, $2 = edgeNow).
func (c *LeaseCoordinator) SweepExpiredLeases(ctx context.Context, edgeNow time.Time) error {
	tx, err := beginTenantTx(ctx, c.db, c.tenantID)
	if err != nil {
		return fmt.Errorf("failed to begin sweep tx: %w", err)
	}
	defer tx.Rollback()

	const query = `
		UPDATE sub_assembly_leases
		SET state = 'OVERRUN_DRAFT',
		    updated_at = $2
		WHERE tenant_id = $1
		  AND state = 'ACTIVE'
		  AND valid_until <= $2
		RETURNING sub_assembly_id, parent_asset_did, assigned_device_id, inspector_id, lease_epoch;
	`
	rows, err := tx.QueryContext(ctx, query, c.tenantID, edgeNow)
	if err != nil {
		return fmt.Errorf("sweep query failed: %w", err)
	}
	defer rows.Close()

	var transitions []transitionRecord
	for rows.Next() {
		var r transitionRecord
		if err := rows.Scan(&r.subAssemblyID, &r.parentAssetDID, &r.assignedDeviceID, &r.inspectorID, &r.leaseEpoch); err != nil {
			return fmt.Errorf("failed to scan sweep row: %w", err)
		}
		transitions = append(transitions, r)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("rows iteration error: %w", err)
	}

	for _, t := range transitions {
		if err := c.provisionOverrunArtifacts(ctx, tx, t, edgeNow); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// provisionOverrunArtifacts is the single authoritative writer for tombstone
// increments and draft work order provisioning across both background sweeps
// and JIT inline overruns.
func (c *LeaseCoordinator) provisionOverrunArtifacts(
	ctx context.Context,
	tx *sql.Tx,
	t transitionRecord,
	edgeNow time.Time,
) error {
	const upsertTombstone = `
		INSERT INTO lease_tombstones (
			tenant_id, organization_id, sub_assembly_id, lease_epoch,
			tombstone_sequence, revoked_device_id, revocation_timestamp,
			authorizing_did, reason
		) VALUES (
			$1, $2, $3, $4, 1, $5, $6, $7, $8
		) ON CONFLICT (tenant_id, sub_assembly_id) DO UPDATE SET
			lease_epoch = EXCLUDED.lease_epoch,
			tombstone_sequence = lease_tombstones.tombstone_sequence + 1,
			revoked_device_id = EXCLUDED.revoked_device_id,
			revocation_timestamp = EXCLUDED.revocation_timestamp,
			authorizing_did = EXCLUDED.authorizing_did,
			reason = EXCLUDED.reason;
	`
	if _, err := tx.ExecContext(ctx, upsertTombstone,
		c.tenantID, c.orgID, t.subAssemblyID, t.leaseEpoch,
		t.assignedDeviceID, edgeNow, c.edgeAuthorityDID, ReasonLeaseOverrun,
	); err != nil {
		return fmt.Errorf("failed to upsert tombstone for %s: %w", t.subAssemblyID, err)
	}

	const insertDraft = `
		INSERT INTO draft_work_orders (
			tenant_id, organization_id, draft_id, sub_assembly_id,
			parent_asset_did, original_inspector_id, lease_epoch,
			state, created_at, updated_at
		) VALUES (
			$1, $2, gen_random_uuid(), $3,
			$4, $5, $6,
			'AWAITING_RECONCILIATION', $7, $7
		) ON CONFLICT (tenant_id, sub_assembly_id, lease_epoch) DO NOTHING;
	`
	if _, err := tx.ExecContext(ctx, insertDraft,
		c.tenantID, c.orgID, t.subAssemblyID, t.parentAssetDID,
		t.inspectorID, t.leaseEpoch, edgeNow,
	); err != nil {
		return fmt.Errorf("failed to create draft envelope for %s: %w", t.subAssemblyID, err)
	}

	return nil
}

// -------------------------------------------------------------------------
// 2. OFFLINE AUTHORITY EPOCH BUNDLE INGESTION
// -------------------------------------------------------------------------

const TagEpochBundle = "INTEGIN-EPOCH-BUNDLE-v1\x00"

// maxBundleFutureSkew caps how far ahead of edge time a bundle may claim to
// be issued before it is treated as pre-dated and rejected.
const maxBundleFutureSkew = 5 * time.Minute

type RevocationEntry struct {
	DeviceID string `json:"device_id"`
	Reason   string `json:"reason"`
}

type EpochBundleManifest struct {
	NewEpoch         uint64            `json:"new_epoch"`
	IssuedAt         time.Time         `json:"issued_at"`
	ValidUntil       time.Time         `json:"valid_until"`
	RevokedDevices   []RevocationEntry `json:"revoked_devices"`
	AuthorityRootDID string            `json:"authority_root_did"`
}

type SignedEpochBundle struct {
	Manifest  EpochBundleManifest `json:"manifest"`
	Signature []byte              `json:"signature"`
}

// Digest packs manifest fields deterministically (explicit field packing, no
// JSON canonicalization dependency): the signer and verifier hash identical
// bytes by construction.
func (m *EpochBundleManifest) Digest() []byte {
	h := sha256.New()
	h.Write([]byte(TagEpochBundle))
	h.Write([]byte(m.AuthorityRootDID))
	_ = binaryWrite(h, m.NewEpoch)
	_ = binaryWrite(h, m.IssuedAt.UnixNano())
	_ = binaryWrite(h, m.ValidUntil.UnixNano())
	_ = binaryWrite(h, uint32(len(m.RevokedDevices)))
	for _, dev := range m.RevokedDevices {
		h.Write([]byte(dev.DeviceID))
		h.Write([]byte(dev.Reason))
	}
	return h.Sum(nil)
}

type EpochIngester struct {
	db         *sql.DB
	tenantID   string
	orgID      string
	rootPubKey crypto.PublicKey
	bundleDir  string
}

func NewEpochIngester(db *sql.DB, tenantID, orgID string, rootPubKey crypto.PublicKey, bundleDir string) *EpochIngester {
	return &EpochIngester{
		db:         db,
		tenantID:   tenantID,
		orgID:      orgID,
		rootPubKey: rootPubKey,
		bundleDir:  filepath.Clean(bundleDir),
	}
}

// resolveBundlePath confines bundlePath inside the configured mount
// directory in two layers: lexical containment first (catches .. escapes
// even for paths that do not exist on disk), then symlink resolution on both
// sides (defeats escapes via linked directories). Missing files fail closed
// with a plain error at the EvalSymlinks stage — distinct from traversal,
// which always yields ErrPathTraversal.
func resolveBundlePath(bundleDir, bundlePath string) (string, error) {
	cleanBaseAbs, err := filepath.Abs(filepath.Clean(bundleDir))
	if err != nil {
		return "", fmt.Errorf("invalid base bundle directory: %w", err)
	}
	targetAbs, err := filepath.Abs(filepath.Clean(bundlePath))
	if err != nil {
		return "", fmt.Errorf("invalid bundle path: %w", err)
	}
	if rel, err := filepath.Rel(cleanBaseAbs, targetAbs); err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("%w: path %s outside %s", ErrPathTraversal, bundlePath, bundleDir)
	}

	realBase, err := filepath.EvalSymlinks(cleanBaseAbs)
	if err != nil {
		return "", fmt.Errorf("invalid base bundle directory: %w", err)
	}
	realTarget, err := filepath.EvalSymlinks(targetAbs)
	if err != nil {
		return "", fmt.Errorf("invalid bundle path: %w", err)
	}
	if rel, err := filepath.Rel(realBase, realTarget); err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("%w: path %s outside %s", ErrPathTraversal, targetAbs, cleanBaseAbs)
	}
	return realTarget, nil
}

// IngestFromPath reads a signed JSON bundle from removable storage, validates
// window sanity and cryptographic provenance, then applies the epoch bump and
// revocations atomically. JSON unmarshal is parse-only: verification runs
// against the explicit field-packed Digest().
func (in *EpochIngester) IngestFromPath(ctx context.Context, bundlePath string, edgeNow time.Time) error {
	targetPath, err := resolveBundlePath(in.bundleDir, bundlePath)
	if err != nil {
		return err
	}

	data, err := os.ReadFile(targetPath)
	if err != nil {
		return fmt.Errorf("failed to read bundle file: %w", err)
	}

	var bundle SignedEpochBundle
	// JSON unmarshal is parse-only here; verification runs against the
	// explicit field-packed Digest(), never the serialized bytes.
	if err := json.Unmarshal(data, &bundle); err != nil {
		return fmt.Errorf("invalid bundle format: %w", err)
	}

	// Window sanity before crypto: malformed windows never reach verification.
	if bundle.Manifest.IssuedAt.After(bundle.Manifest.ValidUntil) {
		return ErrBundleInvalidWindow
	}
	if edgeNow.After(bundle.Manifest.ValidUntil) {
		return ErrBundleExpired
	}
	if bundle.Manifest.IssuedAt.After(edgeNow.Add(maxBundleFutureSkew)) {
		return ErrBundleFutureDated
	}

	// Cryptographic provenance (DER/raw normalized via the domain package).
	digest := bundle.Manifest.Digest()
	if err := domain.VerifySignature(in.rootPubKey, digest, bundle.Signature); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidBundleSignature, err)
	}

	tx, err := beginTenantTx(ctx, in.db, in.tenantID)
	if err != nil {
		return fmt.Errorf("failed to start ingest tx: %w", err)
	}
	defer tx.Rollback()

	// Serializable-equivalent monotonicity: the (tenant_id, epoch) primary key
	// backstops the high-water check against concurrent ingests.
	var currentEpoch uint64
	const selectHighWater = `
		SELECT COALESCE(MAX(epoch), 0)
		FROM authority_epoch_log
		WHERE tenant_id = $1
		FOR UPDATE;
	`
	if err := tx.QueryRowContext(ctx, selectHighWater, in.tenantID).Scan(&currentEpoch); err != nil {
		return fmt.Errorf("failed to read authority epoch high-water: %w", err)
	}

	if bundle.Manifest.NewEpoch <= currentEpoch {
		return fmt.Errorf("%w: bundle epoch %d <= current %d", ErrStaleBundleEpoch, bundle.Manifest.NewEpoch, currentEpoch)
	}

	const insertEpoch = `
		INSERT INTO authority_epoch_log (
			tenant_id, organization_id, epoch,
			authority_root_did, applied_at, bundle_digest
		) VALUES ($1, $2, $3, $4, $5, $6);
	`
	if _, err := tx.ExecContext(ctx, insertEpoch, in.tenantID, in.orgID, bundle.Manifest.NewEpoch, bundle.Manifest.AuthorityRootDID, edgeNow, digest); err != nil {
		return fmt.Errorf("failed to record epoch log: %w", err)
	}

	const insertRevocation = `
		INSERT INTO device_revocation_registry (
			tenant_id, organization_id, device_id,
			revoked_at_epoch, reason, recorded_at
		) VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (tenant_id, device_id) DO UPDATE
		SET revoked_at_epoch = EXCLUDED.revoked_at_epoch,
		    reason = EXCLUDED.reason,
		    recorded_at = EXCLUDED.recorded_at;
	`
	for _, dev := range bundle.Manifest.RevokedDevices {
		if _, err := tx.ExecContext(ctx, insertRevocation, in.tenantID, in.orgID, dev.DeviceID, bundle.Manifest.NewEpoch, dev.Reason, edgeNow); err != nil {
			return fmt.Errorf("failed to register device revocation for %s: %w", dev.DeviceID, err)
		}
	}

	return tx.Commit()
}

// -------------------------------------------------------------------------
// 3. IN-CHAIN MERKLE OVERRIDE COMMIT APPENDER
// -------------------------------------------------------------------------

type MerkleOverrideAppender struct {
	db       *sql.DB
	tenantID string
	orgID    string
}

func NewMerkleOverrideAppender(db *sql.DB, tenantID, orgID string) *MerkleOverrideAppender {
	return &MerkleOverrideAppender{db: db, tenantID: tenantID, orgID: orgID}
}

// CommitEscrowOverride commits an administrative override to the Merkle audit
// chain. Tombstone sequence freshness, lease-epoch equality, quorum validity,
// and draft singularity are all enforced inside one transaction; the returned
// event identity is the commit-order sequence_id.
func (a *MerkleOverrideAppender) CommitEscrowOverride(
	ctx context.Context,
	rec domain.EscrowOverrideRecord,
	tdPubKey crypto.PublicKey,
	ssoPubKey crypto.PublicKey,
	configuredSLA time.Duration,
	edgeArrivalTimestamp time.Time,
) (*domain.MerkleOverrideEvent, error) {
	tx, err := beginTenantTx(ctx, a.db, a.tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to begin override tx: %w", err)
	}
	defer tx.Rollback()

	// 1. Fetch current tombstone state under row lock.
	const getTombstone = `
		SELECT lease_epoch, tombstone_sequence, revocation_timestamp
		FROM lease_tombstones
		WHERE tenant_id = $1 AND sub_assembly_id = $2
		FOR UPDATE;
	`
	var storedEpoch uint64
	var storedSeq uint64
	var tombstoneTime time.Time
	err = tx.QueryRowContext(ctx, getTombstone, a.tenantID, rec.SubAssemblyID).Scan(&storedEpoch, &storedSeq, &tombstoneTime)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: no tombstone recorded for %s", ErrLeaseNotFound, rec.SubAssemblyID)
		}
		return nil, fmt.Errorf("failed to query tombstone: %w", err)
	}

	// 2. Lease-epoch equality: overrides bind exactly one tombstone generation.
	if storedEpoch != rec.LeaseEpoch {
		return nil, fmt.Errorf("%w: override lease_epoch %d != stored tombstone epoch %d", ErrLeaseEpochMismatch, rec.LeaseEpoch, storedEpoch)
	}

	// 3. Quorum against authoritative edge clock and stored sequence.
	if err := rec.ValidateQuorum(tdPubKey, ssoPubKey, configuredSLA, storedSeq, tombstoneTime, edgeArrivalTimestamp); err != nil {
		return nil, fmt.Errorf("quorum validation failed: %w", err)
	}

	// 4. Current high-water authority epoch for dual-epoch evidence.
	var currentAuthEpoch uint64
	err = tx.QueryRowContext(ctx, "SELECT COALESCE(MAX(epoch), 0) FROM authority_epoch_log WHERE tenant_id = $1;", a.tenantID).Scan(&currentAuthEpoch)
	if err != nil {
		return nil, fmt.Errorf("failed to read authority epoch: %w", err)
	}

	// 5. Previous Merkle root; nonzero wrong-length values fail closed.
	var prevHash []byte
	err = tx.QueryRowContext(ctx, "SELECT event_hash FROM merkle_audit_log WHERE tenant_id = $1 ORDER BY sequence_id DESC LIMIT 1 FOR UPDATE;", a.tenantID).Scan(&prevHash)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("failed to query previous merkle root: %w", err)
	}

	var prevRecordHash [32]byte
	if len(prevHash) > 0 {
		if len(prevHash) != 32 {
			return nil, fmt.Errorf("%w: got %d bytes", ErrCorruptedMerkleChain, len(prevHash))
		}
		copy(prevRecordHash[:], prevHash)
	}

	// 6. Event hash binds previous root, payload, both epochs, and commit time.
	payloadDigest := rec.CanonicalDigest()
	var payloadHash [32]byte
	copy(payloadHash[:], payloadDigest)

	h := sha256.New()
	h.Write(prevRecordHash[:])
	h.Write(payloadHash[:])
	_ = binaryWrite(h, currentAuthEpoch)
	_ = binaryWrite(h, rec.LeaseEpoch)
	_ = binaryWrite(h, edgeArrivalTimestamp.UnixNano())
	eventHash := h.Sum(nil)

	const insertEvent = `
		INSERT INTO merkle_audit_log (
			tenant_id, organization_id, sub_assembly_id,
			lease_epoch, authority_epoch, event_type,
			payload_hash, prev_event_hash, event_hash,
			permanent_flag, committed_at
		) VALUES (
			$1, $2, $3,
			$4, $5, $6,
			$7, $8, $9,
			$10, $11
		) RETURNING sequence_id;
	`
	var seqID int64
	if err := tx.QueryRowContext(ctx, insertEvent, a.tenantID, a.orgID, rec.SubAssemblyID, rec.LeaseEpoch, currentAuthEpoch, EventTypeAdminOverride, payloadHash[:], prevRecordHash[:], eventHash, rec.PermanentFlag, edgeArrivalTimestamp).Scan(&seqID); err != nil {
		return nil, fmt.Errorf("failed to append merkle audit event: %w", err)
	}

	// 7. Resolve the draft exactly once: second merges hit zero rows.
	const updateDraft = `
		UPDATE draft_work_orders
		SET state = 'MERGED_VIA_ESCROW',
		    override_id = $3,
		    updated_at = $4
		WHERE tenant_id = $1
		  AND draft_id = $2
		  AND state = 'AWAITING_RECONCILIATION';
	`
	res, err := tx.ExecContext(ctx, updateDraft, a.tenantID, rec.DraftWorkOrderID, rec.OverrideID, edgeArrivalTimestamp)
	if err != nil {
		return nil, fmt.Errorf("failed to update draft work order: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to read draft rows affected: %w", err)
	}
	if rows != 1 {
		return nil, fmt.Errorf("%w: draft %s", ErrDraftNotFound, rec.DraftWorkOrderID)
	}

	// 8. Release the lease so the sub-assembly becomes dispatchable again.
	const releaseLease = `
		UPDATE sub_assembly_leases
		SET state = 'RELEASED',
		    updated_at = $3
		WHERE tenant_id = $1
		  AND sub_assembly_id = $2;
	`
	if _, err := tx.ExecContext(ctx, releaseLease, a.tenantID, rec.SubAssemblyID, edgeArrivalTimestamp); err != nil {
		return nil, fmt.Errorf("failed to release sub-assembly lease: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit override transaction: %w", err)
	}

	return &domain.MerkleOverrideEvent{
		EventID:        fmt.Sprintf("seq-%d", seqID),
		Timestamp:      edgeArrivalTimestamp,
		EventType:      EventTypeAdminOverride,
		AuthorityEpoch: currentAuthEpoch,
		LeaseEpoch:     rec.LeaseEpoch,
		PayloadHash:    payloadHash,
		OverrideEscrow: rec,
		PrevRecordHash: prevRecordHash,
	}, nil
}
