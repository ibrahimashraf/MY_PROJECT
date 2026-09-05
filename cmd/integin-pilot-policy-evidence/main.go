// Command integin-pilot-policy-evidence reports aggregate, non-secret outcome and
// device-state counts for disposable integin-policy matrix transactions. It is
// read-only and deliberately omits all fixture, key, credential, and identifier values.
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func requiredEnvironment(name string) string {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		log.Fatalf("%s is required", name)
	}
	return value
}

func main() {
	databaseURL := requiredEnvironment("INTEGIN_DB_URL")
	tenantID := requiredEnvironment("INTEGIN_TENANT_ID")

	database, err := sql.Open("pgx", databaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()

	context, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := database.PingContext(context); err != nil {
		log.Fatal(err)
	}
	tx, err := database.BeginTx(context, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		log.Fatal(err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(context, "SELECT set_config('integin.tenant_id', $1, true)", tenantID); err != nil {
		log.Fatal(err)
	}

	rows, err := tx.QueryContext(context, `
		SELECT outcome, COUNT(*)
		FROM sync_receipt
		WHERE tenant_id = $1
		  AND transaction_id LIKE 'integin-policy-%'
		GROUP BY outcome
		ORDER BY outcome`, tenantID)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	var applied, duplicate, securityFailure, total int
	for rows.Next() {
		var outcome string
		var count int
		if err := rows.Scan(&outcome, &count); err != nil {
			log.Fatal(err)
		}
		total += count
		switch outcome {
		case "APPLIED":
			applied = count
		case "DUPLICATE":
			duplicate = count
		case "SECURITY_FAILURE":
			securityFailure = count
		}
	}
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}

	var deviceStateRows, maxAcceptedSequence, heldTransactions int
	if err := database.QueryRowContext(context, `
		SELECT COUNT(*), COALESCE(MAX(last_accepted_sequence), 0)
		FROM sync_device_state
		WHERE tenant_id = $1
		  AND device_id IN (
				SELECT DISTINCT device_id
				FROM sync_receipt
				WHERE tenant_id = $1
				  AND transaction_id LIKE 'integin-policy-%'
			)`, tenantID).Scan(&deviceStateRows, &maxAcceptedSequence); err != nil {
		log.Fatal(err)
	}
	if err := database.QueryRowContext(context, `
		SELECT COUNT(*)
		FROM sync_held_transaction
		WHERE tenant_id = $1
		  AND transaction_id LIKE 'integin-policy-%'`, tenantID).Scan(&heldTransactions); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("PILOT_POLICY_EVIDENCE receipts_total=%d applied=%d duplicate=%d security_failure=%d device_state_rows=%d max_accepted_sequence=%d held_transactions=%d\n", total, applied, duplicate, securityFailure, deviceStateRows, maxAcceptedSequence, heldTransactions)
}
