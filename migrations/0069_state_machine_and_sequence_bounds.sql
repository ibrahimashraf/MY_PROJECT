-- integin Hardening (F-14 & F-17): State Machine Closed Vocabularies and Integer Safeties
-- Restricts work_order lifecycle states to formal closed vocabularies and ensures BIGINT upper bound constraints.

BEGIN;

-- Add check constraints on work_order table if not already present
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'work_order_request_state_check'
    ) THEN
        ALTER TABLE work_order
            ADD CONSTRAINT work_order_request_state_check
            CHECK (request_state IN ('draft', 'requested', 'accepted', 'cancelled'));
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'work_order_execution_state_check'
    ) THEN
        ALTER TABLE work_order
            ADD CONSTRAINT work_order_execution_state_check
            CHECK (execution_state IN (
                'ready', 'assigned', 'in_progress', 'suspended',
                'partially_submitted', 'awaiting_client', 'awaiting_review', 'completed'
            ));
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'work_order_commercial_state_check'
    ) THEN
        ALTER TABLE work_order
            ADD CONSTRAINT work_order_commercial_state_check
            CHECK (commercial_state IN (
                'not_ready_for_invoice', 'ready_for_office_confirmation',
                'released_for_invoice', 'invoiced', 'closed'
            ));
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'work_order_certificate_state_check'
    ) THEN
        ALTER TABLE work_order
            ADD CONSTRAINT work_order_certificate_state_check
            CHECK (certificate_state IN (
                'not_started', 'pending_validation', 'partially_issued',
                'issued', 'needs_correction', 'revoked_or_superseded'
            ));
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'work_order_assignment_state_check'
    ) THEN
        ALTER TABLE work_order_assignment
            ADD CONSTRAINT work_order_assignment_state_check
            CHECK (state IN ('active', 'transferred', 'completed', 'revoked'));
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'sync_device_state_sequence_bounds_check'
    ) THEN
        ALTER TABLE sync_device_state
            ADD CONSTRAINT sync_device_state_sequence_bounds_check
            CHECK (last_accepted_sequence >= 0 AND last_accepted_sequence <= 9223372036854775807);
    END IF;
END $$;

COMMIT;
