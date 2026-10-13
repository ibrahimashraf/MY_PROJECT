package migrations

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

const (
	// AdvisoryLockID is a stable 64-bit lock key for INTEGIN schema migrations.
	// 7148858271 corresponds to hash("integin_schema_migrations").
	AdvisoryLockID int64 = 7148858271
)

// Report contains execution telemetry from a migration run.
type Report struct {
	TotalDiscovered int           `json:"total_discovered"`
	AlreadyApplied  int           `json:"already_applied"`
	NewlyApplied    int           `json:"newly_applied"`
	AppliedFiles    []string      `json:"applied_files"`
	Duration        time.Duration `json:"duration"`
}

// EnsureLedgerTable bootstraps the standard schema_migrations table if not already present.
func EnsureLedgerTable(ctx context.Context, db *sql.DB) error {
	const initSQL = `
CREATE TABLE IF NOT EXISTS schema_migrations (
    name TEXT PRIMARY KEY,
    checksum TEXT NOT NULL,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
);`
	_, err := db.ExecContext(ctx, initSQL)
	if err != nil {
		return fmt.Errorf("bootstrap schema_migrations ledger: %w", err)
	}
	return nil
}

// GetAppliedMigrations returns a map of applied migration names to their recorded SHA-256 checksums.
func GetAppliedMigrations(ctx context.Context, db *sql.DB) (map[string]string, error) {
	rows, err := db.QueryContext(ctx, "SELECT name, checksum FROM schema_migrations ORDER BY name;")
	if err != nil {
		return nil, fmt.Errorf("query schema_migrations: %w", err)
	}
	defer rows.Close()

	applied := make(map[string]string)
	for rows.Next() {
		var name, checksum string
		if err := rows.Scan(&name, &checksum); err != nil {
			return nil, fmt.Errorf("scan schema_migrations row: %w", err)
		}
		applied[strings.TrimSpace(name)] = strings.TrimSpace(checksum)
	}
	return applied, rows.Err()
}

// Migrate executes all pending embedded migrations in sequential order with advisory locking.
func Migrate(ctx context.Context, db *sql.DB) (*Report, error) {
	start := time.Now()

	if err := EnsureLedgerTable(ctx, db); err != nil {
		return nil, err
	}

	// Acquire session-level advisory lock to prevent concurrent migration race
	var lockAcquired bool
	if err := db.QueryRowContext(ctx, "SELECT pg_advisory_lock($1);", AdvisoryLockID).Scan(&lockAcquired); err != nil {
		// pg_advisory_lock returns void, scan is no-op, continue
		_ = err
	}
	defer func() {
		_, _ = db.ExecContext(context.Background(), "SELECT pg_advisory_unlock($1);", AdvisoryLockID)
	}()

	files, err := DiscoverUpMigrations()
	if err != nil {
		return nil, err
	}

	applied, err := GetAppliedMigrations(ctx, db)
	if err != nil {
		return nil, err
	}

	// Check for checksum drift on already-applied migrations
	for _, f := range files {
		recordedHash, ok := applied[f.Name]
		if ok && recordedHash != f.Checksum {
			return nil, fmt.Errorf("CHECKSUM DRIFT DETECTED: %s was modified post-apply (recorded: %s, file: %s)", f.Name, recordedHash, f.Checksum)
		}
	}

	report := &Report{
		TotalDiscovered: len(files),
		AlreadyApplied:  len(applied),
		AppliedFiles:    make([]string, 0),
	}

	// Execute pending migrations
	for _, f := range files {
		if _, ok := applied[f.Name]; ok {
			continue // Already applied
		}

		if f.NonTransactional {
			// Non-transactional DDL (e.g. CREATE INDEX CONCURRENTLY) must run outside any transaction block.
			// PostgreSQL wraps multi-statement query strings in an implicit transaction block;
			// therefore, statements must be executed individually.
			statements := splitStatements(f.Content)
			for _, stmt := range statements {
				if strings.TrimSpace(stmt) == "" {
					continue
				}
				if _, err := db.ExecContext(ctx, stmt); err != nil {
					return report, fmt.Errorf("apply non-transactional statement in %s: %w", f.Name, err)
				}
			}
			recordSQL := "INSERT INTO schema_migrations (name, checksum) VALUES ($1, $2);"
			if _, err := db.ExecContext(ctx, recordSQL, f.Name, f.Checksum); err != nil {
				return report, fmt.Errorf("record non-transactional migration %s: %w", f.Name, err)
			}
		} else {
			// Standard migration executes inside an atomic transaction block
			tx, err := db.BeginTx(ctx, nil)
			if err != nil {
				return report, fmt.Errorf("begin tx for %s: %w", f.Name, err)
			}

			if _, err := tx.ExecContext(ctx, f.Content); err != nil {
				_ = tx.Rollback()
				return report, fmt.Errorf("execute migration %s: %w", f.Name, err)
			}

			recordSQL := "INSERT INTO schema_migrations (name, checksum) VALUES ($1, $2);"
			if _, err := tx.ExecContext(ctx, recordSQL, f.Name, f.Checksum); err != nil {
				_ = tx.Rollback()
				return report, fmt.Errorf("record migration %s: %w", f.Name, err)
			}

			if err := tx.Commit(); err != nil {
				return report, fmt.Errorf("commit migration %s: %w", f.Name, err)
			}
		}

		report.NewlyApplied++
		report.AppliedFiles = append(report.AppliedFiles, f.Name)
	}

	report.Duration = time.Since(start)
	return report, nil
}

// RollbackLast reverts the most recent applied migration using its embedded .down.sql pair.
func RollbackLast(ctx context.Context, db *sql.DB) (string, error) {
	if err := EnsureLedgerTable(ctx, db); err != nil {
		return "", err
	}

	var lockAcquired bool
	_ = db.QueryRowContext(ctx, "SELECT pg_advisory_lock($1);", AdvisoryLockID).Scan(&lockAcquired)
	defer func() {
		_, _ = db.ExecContext(context.Background(), "SELECT pg_advisory_unlock($1);", AdvisoryLockID)
	}()

	var lastName string
	err := db.QueryRowContext(ctx, "SELECT name FROM schema_migrations ORDER BY name DESC LIMIT 1;").Scan(&lastName)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("no migrations recorded in ledger to rollback")
		}
		return "", fmt.Errorf("query last applied migration: %w", err)
	}

	downFile, err := DiscoverDownFile(lastName)
	if err != nil {
		return "", err
	}

	isNonTx := strings.Contains(downFile.Content, "CONCURRENTLY") || strings.Contains(downFile.Content, "-- non-transactional")

	if isNonTx {
		statements := splitStatements(downFile.Content)
		for _, stmt := range statements {
			if strings.TrimSpace(stmt) == "" {
				continue
			}
			if _, err := db.ExecContext(ctx, stmt); err != nil {
				return "", fmt.Errorf("execute rollback statement in %s: %w", downFile.Name, err)
			}
		}
		if _, err := db.ExecContext(ctx, "DELETE FROM schema_migrations WHERE name = $1;", lastName); err != nil {
			return "", fmt.Errorf("remove ledger record for %s: %w", lastName, err)
		}
	} else {
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return "", fmt.Errorf("begin tx for rollback of %s: %w", lastName, err)
		}

		if _, err := tx.ExecContext(ctx, downFile.Content); err != nil {
			_ = tx.Rollback()
			return "", fmt.Errorf("execute rollback %s: %w", downFile.Name, err)
		}

		if _, err := tx.ExecContext(ctx, "DELETE FROM schema_migrations WHERE name = $1;", lastName); err != nil {
			_ = tx.Rollback()
			return "", fmt.Errorf("remove ledger record for %s: %w", lastName, err)
		}

		if err := tx.Commit(); err != nil {
			return "", fmt.Errorf("commit rollback for %s: %w", lastName, err)
		}
	}

	return lastName, nil
}

// splitStatements separates SQL content by top-level semicolons, ignoring comments.
func splitStatements(content string) []string {
	var stmts []string
	var buf strings.Builder
	inSingleQuote := false

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "--") {
			continue // skip line comments
		}

		for i := 0; i < len(line); i++ {
			c := line[i]
			if c == '\'' {
				inSingleQuote = !inSingleQuote
			} else if c == ';' && !inSingleQuote {
				buf.WriteByte(c)
				stmt := strings.TrimSpace(buf.String())
				if stmt != "" && stmt != ";" {
					stmts = append(stmts, stmt)
				}
				buf.Reset()
				continue
			}
			buf.WriteByte(c)
		}
		buf.WriteByte('\n')
	}

	if remaining := strings.TrimSpace(buf.String()); remaining != "" && remaining != ";" {
		stmts = append(stmts, remaining)
	}

	return stmts
}
