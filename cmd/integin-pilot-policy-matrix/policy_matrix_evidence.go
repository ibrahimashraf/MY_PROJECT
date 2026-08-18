package main

import (
	"context"
	"database/sql"
	"fmt"
)

// printPolicyMatrixState reports only aggregate counts for the disposable
// device created by the current policy-matrix seed. It never prints the device
// identifier, package identifiers, fixture material, or database configuration.
func printPolicyMatrixState(ctx context.Context, database *sql.DB, tenantID, deviceID, stage string) error {
	transaction, err := database.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return err
	}
	defer transaction.Rollback()
	if _, err := transaction.ExecContext(ctx, "SELECT set_config('integin.tenant_id', $1, true)", tenantID); err != nil {
		return err
	}

	var total, applied, duplicate, securityFailure int
	if err := transaction.QueryRowContext(ctx, `
		SELECT COUNT(*),
		       COUNT(*) FILTER (WHERE outcome = 'APPLIED'),
		       COUNT(*) FILTER (WHERE outcome = 'DUPLICATE'),
		       COUNT(*) FILTER (WHERE outcome = 'SECURITY_FAILURE')
		FROM sync_receipt
		WHERE tenant_id = $1
		  AND device_id = $2
		  AND transaction_id LIKE 'integin-policy-%'`, tenantID, deviceID).Scan(&total, &applied, &duplicate, &securityFailure); err != nil {
		return err
	}

	var deviceStateRows, maxAcceptedSequence, heldTransactions int
	if err := transaction.QueryRowContext(ctx, `
		SELECT COUNT(*), COALESCE(MAX(last_accepted_sequence), 0)
		FROM sync_device_state
		WHERE tenant_id = $1 AND device_id = $2`, tenantID, deviceID).Scan(&deviceStateRows, &maxAcceptedSequence); err != nil {
		return err
	}
	if err := transaction.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM sync_held_transaction
		WHERE tenant_id = $1
		  AND device_id = $2
		  AND transaction_id LIKE 'integin-policy-%'`, tenantID, deviceID).Scan(&heldTransactions); err != nil {
		return err
	}

	fmt.Printf("PILOT_POLICY_MATRIX_STATE stage=%s receipts_total=%d applied=%d duplicate=%d security_failure=%d device_state_rows=%d max_accepted_sequence=%d held_transactions=%d\n", stage, total, applied, duplicate, securityFailure, deviceStateRows, maxAcceptedSequence, heldTransactions)
	return transaction.Commit()
}
