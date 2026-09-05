package workorderpg

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"database/sql/driver"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
	"strings"

	"integin/internal/domain/workorder"
	"integin/internal/shared/pgtx"
)

var (
	ErrNilDB                           = errors.New("work-order postgres repository requires a database")
	ErrUnsupported                     = errors.New("work-order postgres operation is not implemented in this slice")
	ErrStaleRevision                   = errors.New("work-order revision is stale")
	ErrIdempotencyMismatch             = errors.New("work-order idempotency payload mismatch")
	ErrInspectionAlreadySubmitted      = errors.New("work-order inspection already submitted")
	ErrInspectionMembershipUnavailable = errors.New("work-order inspection membership validator is not configured")
)

type InspectionMembershipValidator interface {
	ValidatePartialSubmission(context.Context, workorder.ActorContext, string, string, []string) error
}

type Repository struct {
	db                   *sql.DB
	inspectionMembership InspectionMembershipValidator
}

func NewRepository(db *sql.DB, validators ...InspectionMembershipValidator) (*Repository, error) {
	if db == nil {
		return nil, ErrNilDB
	}
	var membership InspectionMembershipValidator
	if len(validators) > 0 {
		membership = validators[0]
	}
	return &Repository{db: db, inspectionMembership: membership}, nil
}

func (r *Repository) begin(ctx context.Context, actor workorder.ActorContext) (*sql.Tx, error) {
	if err := actor.Validate(); err != nil {
		return nil, err
	}
	return pgtx.BeginScope(ctx, r.db, actor.TenantID, actor.OrganizationID)
}

func (r *Repository) GetWorkOrder(ctx context.Context, actor workorder.ActorContext, workOrderID string) (workorder.WorkOrder, error) {
	tx, err := r.begin(ctx, actor)
	if err != nil {
		return workorder.WorkOrder{}, err
	}
	defer tx.Rollback()
	return loadOrder(ctx, tx, actor, workOrderID, false)
}

func (r *Repository) GetAssignment(ctx context.Context, actor workorder.ActorContext, assignmentID string) (workorder.Assignment, error) {
	tx, err := r.begin(ctx, actor)
	if err != nil {
		return workorder.Assignment{}, err
	}
	defer tx.Rollback()
	return loadAssignment(ctx, tx, actor, assignmentID, false)
}

func (r *Repository) FindOperationReceipt(ctx context.Context, actor workorder.ActorContext, idempotencyKey string) (workorder.MutationReceipt, bool, error) {
	tx, err := r.begin(ctx, actor)
	if err != nil {
		return workorder.MutationReceipt{}, false, err
	}
	defer tx.Rollback()
	_, receipt, found, err := findReceipt(ctx, tx, actor, idempotencyKey, "")
	return receipt, found, err
}

func (r *Repository) CreateRequest(ctx context.Context, command workorder.CreateRequestCommand) (workorder.MutationReceipt, error) {
	if err := workorder.ValidateCreateRequestCommand(command); err != nil {
		return workorder.MutationReceipt{}, err
	}
	requestHash, err := hashPayload(command.WorkOrder)
	if err != nil {
		return workorder.MutationReceipt{}, err
	}
	tx, receipt, found, err := r.startMutation(ctx, command.Actor, command.Operation, requestHash)
	if err != nil || found {
		if found {
			_ = tx.Rollback()
		}
		return receipt, err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO work_order (id, tenant_id, organization_id, client_id, job_number,
			request_state, execution_state, commercial_state, certificate_state, revision, created_by, updated_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $11)`,
		command.WorkOrder.ID, command.WorkOrder.TenantID, command.WorkOrder.OrganizationID,
		command.WorkOrder.ClientID, command.WorkOrder.JobNumber,
		command.WorkOrder.RequestState, command.WorkOrder.ExecutionState,
		command.WorkOrder.CommercialState, command.WorkOrder.CertificateState,
		command.WorkOrder.Revision, command.Actor.ActorID)
	if err != nil {
		return receipt, err
	}
	receipt = acceptedReceipt(command.Actor, command.Operation, command.WorkOrder.ID, command.WorkOrder.Revision)
	if err := finishMutation(ctx, tx, command.Actor, command.Operation, requestHash, "create_request", command.WorkOrder.ID, command.Operation.ExpectedRevision, command.WorkOrder.Revision, receipt); err != nil {
		return receipt, err
	}
	return receipt, tx.Commit()
}

func (r *Repository) AssignScope(ctx context.Context, command workorder.AssignScopeCommand) (workorder.MutationReceipt, error) {
	requestHash, err := hashPayload(command)
	if err != nil {
		return workorder.MutationReceipt{}, err
	}
	tx, receipt, found, err := r.startMutation(ctx, command.Actor, command.Operation, requestHash)
	if err != nil || found {
		if found {
			_ = tx.Rollback()
		}
		return receipt, err
	}
	defer tx.Rollback()
	order, err := loadOrder(ctx, tx, command.Actor, command.WorkOrderID, true)
	if err != nil {
		return receipt, err
	}
	if err := workorder.ValidateAssignScopeCommand(command, order); err != nil {
		return receipt, err
	}
	if command.Operation.ExpectedRevision != order.Revision {
		return receipt, ErrStaleRevision
	}
	if err := verifyScopeIDs(ctx, tx, command.Actor, order, command.Assignment.ScopeItemIDs); err != nil {
		return receipt, err
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO work_order_assignment
			(id, tenant_id, organization_id, work_order_id, inspector_id, state, revision, effective_from, effective_until, created_by, updated_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $10)`,
		command.Assignment.ID, command.Actor.TenantID, command.Actor.OrganizationID, command.WorkOrderID,
		command.Assignment.InspectorID, command.Assignment.State, command.Assignment.Revision,
		command.Assignment.EffectiveFrom, command.Assignment.EffectiveUntil, command.Actor.ActorID)
	if err != nil {
		return receipt, err
	}
	if err := insertAssignmentScope(ctx, tx, command.Actor, command.Assignment.ID, command.Assignment.ScopeItemIDs); err != nil {
		return receipt, err
	}
	newRevision, err := bumpOrder(ctx, tx, command.Actor, order, workorder.ExecutionAssigned, workorder.ExecutionAssigned)
	if err != nil {
		return receipt, err
	}
	receipt = acceptedReceipt(command.Actor, command.Operation, order.ID, newRevision)
	if err := finishMutation(ctx, tx, command.Actor, command.Operation, requestHash, "assign_scope", order.ID, command.Operation.ExpectedRevision, newRevision, receipt); err != nil {
		return receipt, err
	}
	return receipt, tx.Commit()
}

func (r *Repository) TransitionExecution(ctx context.Context, command workorder.TransitionExecutionCommand) (workorder.MutationReceipt, error) {
	requestHash, err := hashPayload(command)
	if err != nil {
		return workorder.MutationReceipt{}, err
	}
	tx, receipt, found, err := r.startMutation(ctx, command.Actor, command.Operation, requestHash)
	if err != nil || found {
		if found {
			_ = tx.Rollback()
		}
		return receipt, err
	}
	defer tx.Rollback()
	order, err := loadOrder(ctx, tx, command.Actor, command.WorkOrderID, true)
	if err != nil {
		return receipt, err
	}
	if err := workorder.ValidateTransitionExecutionCommand(command, order); err != nil {
		return receipt, err
	}
	if command.Operation.ExpectedRevision != order.Revision {
		return receipt, ErrStaleRevision
	}
	newRevision, err := bumpOrder(ctx, tx, command.Actor, order, command.To, command.To)
	if err != nil {
		return receipt, err
	}
	if err := insertStateEvent(ctx, tx, command.Actor, order.ID, command.Actor.ActorID, "execution_transition", command.Operation.OperationID, command.From, command.To); err != nil {
		return receipt, err
	}
	receipt = acceptedReceipt(command.Actor, command.Operation, order.ID, newRevision)
	if err := finishMutation(ctx, tx, command.Actor, command.Operation, requestHash, "transition_execution", order.ID, command.Operation.ExpectedRevision, newRevision, receipt); err != nil {
		return receipt, err
	}
	return receipt, tx.Commit()
}

func (r *Repository) SubmitPartial(ctx context.Context, command workorder.SubmitPartialCommand) (workorder.MutationReceipt, error) {
	requestHash, err := hashPayload(command)
	if err != nil {
		return workorder.MutationReceipt{}, err
	}
	tx, receipt, found, err := r.startMutation(ctx, command.Actor, command.Operation, requestHash)
	if err != nil || found {
		if found {
			_ = tx.Rollback()
		}
		return receipt, err
	}
	defer tx.Rollback()
	order, err := loadOrder(ctx, tx, command.Actor, command.WorkOrderID, true)
	if err != nil {
		return receipt, err
	}
	assignment, err := loadAssignment(ctx, tx, command.Actor, command.AssignmentID, true)
	if err != nil {
		return receipt, err
	}
	if err := workorder.ValidateSubmitPartialCommand(command, order); err != nil {
		return receipt, err
	}
	if assignment.WorkOrderID != order.ID || assignment.OrganizationID != order.OrganizationID {
		return receipt, workorder.ErrInvalidScope
	}
	if command.Operation.ExpectedRevision != order.Revision {
		return receipt, ErrStaleRevision
	}
	if r.inspectionMembership == nil {
		return receipt, ErrInspectionMembershipUnavailable
	}
	var transactionalMembership transactionalInspectionMembershipValidator
	var memberships []inspectionMembership
	if membership, ok := r.inspectionMembership.(transactionalInspectionMembershipValidator); ok {
		transactionalMembership = membership
		memberships, err = membership.ValidatePartialSubmissionTx(ctx, tx, command.Actor, order.ID, assignment.ID, command.InspectionIDs)
		if err != nil {
			return receipt, err
		}
	} else if err := r.inspectionMembership.ValidatePartialSubmission(ctx, command.Actor, order.ID, assignment.ID, command.InspectionIDs); err != nil {
		return receipt, err
	}
	var alreadySubmitted bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM work_order_submission_segment WHERE tenant_id=$1 AND organization_id=$2 AND work_order_id=$3 AND state <> 'rejected' AND inspection_ids ?| $4::text[])`, command.Actor.TenantID, command.Actor.OrganizationID, order.ID, pqStringArray(command.InspectionIDs)).Scan(&alreadySubmitted); err != nil {
		return receipt, err
	}
	if alreadySubmitted {
		return receipt, ErrInspectionAlreadySubmitted
	}
	inspectionJSON, err := json.Marshal(command.InspectionIDs)
	if err != nil {
		return receipt, err
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO work_order_submission_segment
			(id, tenant_id, organization_id, work_order_id, assignment_id, submitted_by, state, revision, inspection_ids)
		VALUES ($1, $2, $3, $4, $5, $6, 'submitted', $7, $8)`,
		command.Operation.OperationID, command.Actor.TenantID, command.Actor.OrganizationID,
		order.ID, assignment.ID, command.Actor.ActorID, order.Revision+1, inspectionJSON)
	if err != nil {
		return receipt, err
	}
	if transactionalMembership != nil {
		if err := transactionalMembership.RecordPartialSubmissionTx(ctx, tx, command.Actor, command.Operation.OperationID, order.ID, assignment.ID, memberships); err != nil {
			return receipt, err
		}
	}
	newRevision, err := bumpOrder(ctx, tx, command.Actor, order, workorder.ExecutionPartiallySubmitted, workorder.ExecutionPartiallySubmitted)
	if err != nil {
		return receipt, err
	}
	receipt = acceptedReceipt(command.Actor, command.Operation, order.ID, newRevision)
	if err := finishMutation(ctx, tx, command.Actor, command.Operation, requestHash, "submit_partial", order.ID, command.Operation.ExpectedRevision, newRevision, receipt); err != nil {
		return receipt, err
	}
	return receipt, tx.Commit()
}

func (r *Repository) ReassignScope(ctx context.Context, command workorder.ReassignScopeCommand) (workorder.MutationReceipt, error) {
	requestHash, err := hashPayload(command)
	if err != nil {
		return workorder.MutationReceipt{}, err
	}
	tx, receipt, found, err := r.startMutation(ctx, command.Actor, command.Operation, requestHash)
	if err != nil || found {
		if found {
			_ = tx.Rollback()
		}
		return receipt, err
	}
	defer tx.Rollback()
	order, err := loadOrder(ctx, tx, command.Actor, command.WorkOrderID, true)
	if err != nil {
		return receipt, err
	}
	if err := workorder.ValidateReassignScopeCommand(command, order); err != nil {
		return receipt, err
	}
	previous, err := loadAssignment(ctx, tx, command.Actor, command.Previous.ID, true)
	if err != nil {
		return receipt, err
	}
	if previous.WorkOrderID != order.ID || previous.State != workorder.AssignmentActive || previous.Revision != command.Previous.Revision {
		return receipt, ErrStaleRevision
	}

	if command.Operation.ExpectedRevision != order.Revision {
		return receipt, ErrStaleRevision
	}
	if err := verifyScopeIDs(ctx, tx, command.Actor, order, command.ScopeItemIDs); err != nil {
		return receipt, err
	}
	result, err := tx.ExecContext(ctx, `UPDATE work_order_assignment SET state='transferred', updated_by=$1, updated_at=now() WHERE id=$2 AND tenant_id=$3 AND organization_id=$4 AND work_order_id=$5 AND revision=$6 AND state='active'`, command.Actor.ActorID, command.Previous.ID, command.Actor.TenantID, command.Actor.OrganizationID, order.ID, previous.Revision)
	if err != nil {
		return receipt, err
	}
	updated, err := result.RowsAffected()
	if err != nil || updated != 1 {
		return receipt, ErrStaleRevision
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO work_order_assignment
			(id, tenant_id, organization_id, work_order_id, inspector_id, state, revision, effective_from, effective_until, created_by, updated_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $10)`,
		command.Replacement.ID, command.Actor.TenantID, command.Actor.OrganizationID, order.ID,
		command.Replacement.InspectorID, command.Replacement.State, command.Replacement.Revision,
		command.Replacement.EffectiveFrom, command.Replacement.EffectiveUntil, command.Actor.ActorID)
	if err != nil {
		return receipt, err
	}
	if err := insertAssignmentScope(ctx, tx, command.Actor, command.Replacement.ID, command.ScopeItemIDs); err != nil {
		return receipt, err
	}
	newRevision, err := bumpOrder(ctx, tx, command.Actor, order, workorder.ExecutionAssigned, workorder.ExecutionAssigned)
	if err != nil {
		return receipt, err
	}
	receipt = acceptedReceipt(command.Actor, command.Operation, order.ID, newRevision)
	if err := finishMutation(ctx, tx, command.Actor, command.Operation, requestHash, "reassign_scope", order.ID, command.Operation.ExpectedRevision, newRevision, receipt); err != nil {
		return receipt, err
	}
	return receipt, tx.Commit()
}

func (r *Repository) ReconcileProvisional(ctx context.Context, command workorder.ReconcileProvisionalCommand) (workorder.MutationReceipt, error) {
	if err := workorder.ValidateReconcileProvisionalCommand(command); err != nil {
		return workorder.MutationReceipt{}, err
	}
	requestHash, err := hashPayload(command)
	if err != nil {
		return workorder.MutationReceipt{}, err
	}
	tx, receipt, found, err := r.startMutation(ctx, command.Actor, command.Operation, requestHash)
	if err != nil || found {
		if found {
			_ = tx.Rollback()
		}
		return receipt, err
	}
	defer tx.Rollback()
	candidateFingerprint, err := hashPayload(command.Record)
	if err != nil {
		return receipt, err
	}
	var existingFingerprint string
	lookupErr := tx.QueryRowContext(ctx, `
			SELECT candidate_fingerprint
			FROM work_order_provisional_record
			WHERE tenant_id=$1 AND organization_id=$2 AND local_id=$3
			FOR UPDATE`, command.Actor.TenantID, command.Actor.OrganizationID, command.Record.LocalID).Scan(&existingFingerprint)
	switch lookupErr {
	case nil:
		if err := workorder.ValidateProvisionalFingerprint(existingFingerprint, candidateFingerprint); err != nil {
			return receipt, err
		}
	case sql.ErrNoRows:
	default:
		return receipt, lookupErr
	}
	result, err := tx.ExecContext(ctx, `
			INSERT INTO work_order_provisional_record

			(id, tenant_id, organization_id, local_id, record_kind, work_order_id, client_id, canonical_id, reconcile_state, candidate_fingerprint, reconciled_at, reconciled_by)
		VALUES ($1, $2, $3, $4, $5, $6, NULLIF($7,''), NULLIF($8,''), $9, $10, CASE WHEN $9 IN ('matched','created','conflict','rejected') THEN now() ELSE NULL END, CASE WHEN $9 IN ('matched','created','conflict','rejected') THEN $11 ELSE NULL END)
					ON CONFLICT (tenant_id, organization_id, local_id) DO UPDATE SET
				canonical_id=EXCLUDED.canonical_id, reconcile_state=EXCLUDED.reconcile_state,
				reconciled_at=EXCLUDED.reconciled_at, reconciled_by=EXCLUDED.reconciled_by
			WHERE work_order_provisional_record.candidate_fingerprint = EXCLUDED.candidate_fingerprint`,

		command.Operation.OperationID, command.Actor.TenantID, command.Actor.OrganizationID,
		command.Record.LocalID, command.Record.Kind, command.Record.WorkOrderID, command.Record.ClientID,
		command.CanonicalID, command.Outcome, candidateFingerprint, command.Actor.ActorID)
	if err != nil {
		return receipt, err
	}
	updated, err := result.RowsAffected()
	if err != nil {
		return receipt, err
	}
	if updated != 1 {
		return receipt, workorder.ErrProvisionalConflict
	}
	receipt = acceptedReceipt(command.Actor, command.Operation, command.Record.WorkOrderID, command.Operation.ExpectedRevision)

	if err := finishMutation(ctx, tx, command.Actor, command.Operation, requestHash, "reconcile_provisional", command.Record.WorkOrderID, command.Operation.ExpectedRevision, command.Operation.ExpectedRevision, receipt); err != nil {
		return receipt, err
	}
	return receipt, tx.Commit()
}

func validateCertificateInspections(ctx context.Context, tx *sql.Tx, actor workorder.ActorContext, workOrderID string, inspectionIDs []string) error {
	if len(inspectionIDs) == 0 {
		return nil
	}

	// Single query using ANY to fetch all inspections at once (fixes N+1)
	// Instead of one SELECT per inspectionID, we do one SELECT with id = ANY($4)
	rows, err := tx.QueryContext(ctx, `
		SELECT id, lifecycle_state, finalization_state
		FROM inspection_record
		WHERE tenant_id=$1 AND organization_id=$2 AND work_order_id=$3 AND id = ANY($4)
		FOR UPDATE`, actor.TenantID, actor.OrganizationID, workOrderID, pqStringArray(inspectionIDs))
	if err != nil {
		return err
	}
	defer rows.Close()

	// Build a map of inspection results
	inspectionMap := make(map[string]struct {
		lifecycleState  string
		finalizationState string
	})
	for rows.Next() {
		var id, lifecycleState, finalizationState string
		if err := rows.Scan(&id, &lifecycleState, &finalizationState); err != nil {
			return err
		}
		inspectionMap[id] = struct {
			lifecycleState  string
			finalizationState string
		}{lifecycleState: lifecycleState, finalizationState: finalizationState}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	// Validate each inspection from the map (no more DB queries)
	for _, inspectionID := range inspectionIDs {
		inv, ok := inspectionMap[inspectionID]
		if !ok {
			return workorder.ErrInvalidScope
		}
		if err := workorder.ValidateCertificateInspectionState(inv.lifecycleState, inv.finalizationState); err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) RequestCertificateValidation(ctx context.Context, command workorder.RequestCertificateValidationCommand) (workorder.MutationReceipt, error) {
	requestHash, err := hashPayload(command)
	if err != nil {
		return workorder.MutationReceipt{}, err
	}
	tx, receipt, found, err := r.startMutation(ctx, command.Actor, command.Operation, requestHash)
	if err != nil || found {
		if found {
			_ = tx.Rollback()
		}
		return receipt, err
	}
	defer tx.Rollback()
	order, err := loadOrder(ctx, tx, command.Actor, command.WorkOrderID, true)
	if err != nil {
		return receipt, err
	}
	if err := workorder.ValidateCertificateValidationCommand(command, order); err != nil {
		return receipt, err
	}
	if err := validateCertificateInspections(ctx, tx, command.Actor, command.WorkOrderID, command.InspectionIDs); err != nil {
		return receipt, err
	}
	if command.Operation.ExpectedRevision != order.Revision {

		return receipt, ErrStaleRevision
	}
	newRevision, err := bumpOrder(ctx, tx, command.Actor, order, order.ExecutionState, order.ExecutionState)
	if err != nil {
		return receipt, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE work_order SET certificate_state='pending_validation', updated_by=$1, updated_at=now() WHERE id=$2 AND tenant_id=$3 AND organization_id=$4`, command.Actor.ActorID, order.ID, command.Actor.TenantID, command.Actor.OrganizationID); err != nil {
		return receipt, err
	}
	if err := insertStateEvent(ctx, tx, command.Actor, order.ID, command.Actor.ActorID, "certificate_validation_requested", command.Operation.OperationID, order.CertificateState, workorder.CertificatePendingValidation); err != nil {
		return receipt, err
	}
	receipt = acceptedReceipt(command.Actor, command.Operation, order.ID, newRevision)
	if err := finishMutation(ctx, tx, command.Actor, command.Operation, requestHash, "request_certificate_validation", order.ID, command.Operation.ExpectedRevision, newRevision, receipt); err != nil {
		return receipt, err
	}
	return receipt, tx.Commit()
}

func (r *Repository) AddEvidenceReference(ctx context.Context, command workorder.AddEvidenceReferenceCommand) (workorder.MutationReceipt, error) {
	requestHash, err := hashPayload(command)
	if err != nil {
		return workorder.MutationReceipt{}, err
	}
	tx, receipt, found, err := r.startMutation(ctx, command.Actor, command.Operation, requestHash)
	if err != nil || found {
		if found {
			_ = tx.Rollback()
		}
		return receipt, err
	}
	defer tx.Rollback()
	order, err := loadOrder(ctx, tx, command.Actor, command.Evidence.WorkOrderID, true)
	if err != nil {
		return receipt, err
	}
	if err := workorder.ValidateAddEvidenceReferenceCommand(command, order); err != nil {
		return receipt, err
	}
	if command.Operation.ExpectedRevision != order.Revision {
		return receipt, ErrStaleRevision
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO work_order_evidence
			(id, tenant_id, organization_id, work_order_id, content_hash, reference_url, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		command.Evidence.ID, command.Actor.TenantID, command.Actor.OrganizationID,
		command.Evidence.WorkOrderID, command.Evidence.ContentHash, command.Evidence.ReferenceURL, command.Actor.ActorID); err != nil {
		return receipt, err
	}
	receipt = acceptedReceipt(command.Actor, command.Operation, order.ID, order.Revision)
	if err := finishMutation(ctx, tx, command.Actor, command.Operation, requestHash, "add_evidence_reference", order.ID, command.Operation.ExpectedRevision, order.Revision, receipt); err != nil {
		return receipt, err
	}
	return receipt, tx.Commit()
}

func (r *Repository) startMutation(ctx context.Context, actor workorder.ActorContext, operation workorder.OperationMeta, requestHash string) (*sql.Tx, workorder.MutationReceipt, bool, error) {
	if err := operation.Validate(); err != nil {
		return nil, workorder.MutationReceipt{}, false, err
	}
	tx, err := r.begin(ctx, actor)
	if err != nil {
		return nil, workorder.MutationReceipt{}, false, err
	}
	storedHash, receipt, found, err := findReceipt(ctx, tx, actor, operation.IdempotencyKey, requestHash)
	if err != nil {
		_ = tx.Rollback()
		return nil, workorder.MutationReceipt{}, false, err
	}
	if found {
		if storedHash != requestHash {
			_ = tx.Rollback()
			return nil, workorder.MutationReceipt{}, false, ErrIdempotencyMismatch
		}
		return tx, receipt, true, nil
	}
	return tx, receipt, false, nil
}

func findReceipt(ctx context.Context, tx *sql.Tx, actor workorder.ActorContext, key, _ string) (string, workorder.MutationReceipt, bool, error) {
	var hash string
	var raw []byte
	err := tx.QueryRowContext(ctx, `SELECT request_hash, receipt FROM work_order_operation WHERE idempotency_key=$1 AND tenant_id=$2 AND organization_id=$3 FOR UPDATE`, key, actor.TenantID, actor.OrganizationID).Scan(&hash, &raw)
	if errors.Is(err, sql.ErrNoRows) {
		return "", workorder.MutationReceipt{}, false, nil
	}
	if err != nil {
		return "", workorder.MutationReceipt{}, false, err
	}
	var receipt workorder.MutationReceipt
	if err := json.Unmarshal(raw, &receipt); err != nil {
		return "", receipt, false, err
	}
	return hash, receipt, true, nil
}

func finishMutation(ctx context.Context, tx *sql.Tx, actor workorder.ActorContext, operation workorder.OperationMeta, requestHash, operationType, aggregateID string, expected, resulting int64, receipt workorder.MutationReceipt) error {
	raw, err := json.Marshal(receipt)
	if err != nil {
		return err
	}
	payloadHash := requestHash
	if len(payloadHash) != 64 {
		sum := sha256.Sum256(raw)
		payloadHash = hex.EncodeToString(sum[:])
	}
	signature := "sig_system_generated"
	signingKeyID := "key_system_internal"
	_, err = tx.ExecContext(ctx, `
		INSERT INTO work_order_operation
			(id, tenant_id, organization_id, operation_id, idempotency_key, request_hash, operation_type, aggregate_id, expected_revision, resulting_revision, status, receipt, payload_hash, signature, signing_key_id, completed_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,now())`,
		operation.OperationID, actor.TenantID, actor.OrganizationID, operation.OperationID,
		operation.IdempotencyKey, requestHash, operationType, aggregateID, expected, resulting,
		workorder.ReceiptAccepted, raw, payloadHash, signature, signingKeyID)
	return err
}

func acceptedReceipt(actor workorder.ActorContext, operation workorder.OperationMeta, orderID string, revision int64) workorder.MutationReceipt {
	return workorder.MutationReceipt{OperationID: operation.OperationID, IdempotencyKey: operation.IdempotencyKey, TenantID: actor.TenantID, WorkOrderID: orderID, Revision: revision, Status: workorder.ReceiptAccepted}
}

func loadOrder(ctx context.Context, tx *sql.Tx, actor workorder.ActorContext, id string, forUpdate bool) (workorder.WorkOrder, error) {
	query := `SELECT id,job_number,tenant_id,organization_id,client_id,request_state,execution_state,commercial_state,certificate_state,revision FROM work_order WHERE id=$1 AND tenant_id=$2 AND organization_id=$3`
	if forUpdate {
		query += " FOR UPDATE"
	}
	var order workorder.WorkOrder
	err := tx.QueryRowContext(ctx, query, id, actor.TenantID, actor.OrganizationID).Scan(&order.ID, &order.JobNumber, &order.TenantID, &order.OrganizationID, &order.ClientID, &order.RequestState, &order.ExecutionState, &order.CommercialState, &order.CertificateState, &order.Revision)
	return order, err
}

func loadAssignment(ctx context.Context, tx *sql.Tx, actor workorder.ActorContext, id string, forUpdate bool) (workorder.Assignment, error) {
	query := `SELECT id,tenant_id,organization_id,work_order_id,inspector_id,state,revision,effective_from,effective_until FROM work_order_assignment WHERE id=$1 AND tenant_id=$2 AND organization_id=$3`
	if forUpdate {
		query += " FOR UPDATE"
	}
	var assignment workorder.Assignment
	err := tx.QueryRowContext(ctx, query, id, actor.TenantID, actor.OrganizationID).Scan(&assignment.ID, &assignment.TenantID, &assignment.OrganizationID, &assignment.WorkOrderID, &assignment.InspectorID, &assignment.State, &assignment.Revision, &assignment.EffectiveFrom, &assignment.EffectiveUntil)
	if err != nil {
		return assignment, err
	}
	rows, err := tx.QueryContext(ctx, `SELECT scope_item_id FROM work_order_assignment_scope WHERE assignment_id=$1 AND tenant_id=$2 AND organization_id=$3 ORDER BY scope_item_id`, id, actor.TenantID, actor.OrganizationID)
	if err != nil {
		return assignment, err
	}
	defer rows.Close()
	for rows.Next() {
		var scopeID string
		if err := rows.Scan(&scopeID); err != nil {
			return assignment, err
		}
		assignment.ScopeItemIDs = append(assignment.ScopeItemIDs, scopeID)
	}
	return assignment, rows.Err()
}

func verifyScopeIDs(ctx context.Context, tx *sql.Tx, actor workorder.ActorContext, order workorder.WorkOrder, ids []string) error {
	var count int
	if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM work_order_scope_item WHERE tenant_id=$1 AND organization_id=$2 AND work_order_id=$3 AND id = ANY($4)`, actor.TenantID, actor.OrganizationID, order.ID, pqStringArray(ids)).Scan(&count); err != nil {
		return err
	}
	if count != len(ids) {
		return workorder.ErrInvalidScope
	}
	return nil
}

func insertAssignmentScope(ctx context.Context, tx *sql.Tx, actor workorder.ActorContext, assignmentID string, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	// Bulk insert using unnest to avoid N+1 write amplification
	// Instead of one INSERT per id, we do one INSERT with unnest
	_, err := tx.ExecContext(ctx, `
		INSERT INTO work_order_assignment_scope (tenant_id, organization_id, assignment_id, scope_item_id)
		SELECT $1::text, $2::text, $3::text, unnest($4::text[])
	`, actor.TenantID, actor.OrganizationID, assignmentID, pqStringArray(ids))
	return err
}

func bumpOrder(ctx context.Context, tx *sql.Tx, actor workorder.ActorContext, order workorder.WorkOrder, state, _ workorder.ExecutionState) (int64, error) {
	newRevision := order.Revision + 1
	result, err := tx.ExecContext(ctx, `UPDATE work_order SET execution_state=$1, revision=$2, updated_by=$3, updated_at=now() WHERE id=$4 AND tenant_id=$5 AND organization_id=$6 AND revision=$7`, state, newRevision, actor.ActorID, order.ID, actor.TenantID, actor.OrganizationID, order.Revision)
	if err != nil {
		return 0, err
	}
	rows, err := result.RowsAffected()
	if err != nil || rows != 1 {
		return 0, ErrStaleRevision
	}
	return newRevision, nil
}

func insertStateEvent(ctx context.Context, tx *sql.Tx, actor workorder.ActorContext, orderID, actorID, eventType, operationID string, previous, next any) error {
	previousJSON, err := json.Marshal(previous)
	if err != nil {
		return err
	}
	nextJSON, err := json.Marshal(next)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO work_order_state_event (id,tenant_id,organization_id,work_order_id,actor_id,event_type,previous_state,new_state,operation_id) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`, operationID+"-event", actor.TenantID, actor.OrganizationID, orderID, actorID, eventType, previousJSON, nextJSON, operationID)
	return err
}

func hashPayload(value any) (string, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	// Canonicalize via generic map/array sorting to guarantee field-order independence
	var generic any
	if err := json.Unmarshal(encoded, &generic); err != nil {
		return "", err
	}
	canonicalBytes, err := canonicalJSONBytes(generic)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(canonicalBytes)
	return hex.EncodeToString(sum[:]), nil
}

func canonicalJSONBytes(v any) ([]byte, error) {
	switch val := v.(type) {
	case map[string]any:
		keys := make([]string, 0, len(val))
		for k := range val {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		var buf bytes.Buffer
		buf.WriteByte('{')
		for i, k := range keys {
			if i > 0 {
				buf.WriteByte(',')
			}
			kb, _ := json.Marshal(k)
			buf.Write(kb)
			buf.WriteByte(':')
			vb, err := canonicalJSONBytes(val[k])
			if err != nil {
				return nil, err
			}
			buf.Write(vb)
		}
		buf.WriteByte('}')
		return buf.Bytes(), nil
	case []any:
		var buf bytes.Buffer
		buf.WriteByte('[')
		for i, item := range val {
			if i > 0 {
				buf.WriteByte(',')
			}
			ib, err := canonicalJSONBytes(item)
			if err != nil {
				return nil, err
			}
			buf.Write(ib)
		}
		buf.WriteByte(']')
		return buf.Bytes(), nil
	default:
		return json.Marshal(val)
	}
}

func pqStringArray(values []string) interface{ driver.Valuer } {
	return stringArray(values)
}

type stringArray []string

func (a stringArray) Value() (driver.Value, error) {
	quoted := make([]string, len(a))
	for i, value := range a {
		escaped := strings.ReplaceAll(value, `\`, `\\`)
		escaped = strings.ReplaceAll(escaped, `"`, `\"`)
		quoted[i] = `"` + escaped + `"`
	}
	return "{" + strings.Join(quoted, ",") + "}", nil
}
