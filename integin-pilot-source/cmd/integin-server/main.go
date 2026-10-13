// Command integin-server is the universal INTEGIN server binary.
//
// Deployment topology is selected with --mode=cloud|sovereign|airgap
// (default cloud), which presets INTEGIN_DEPLOY_MODE before booting the shared
// server logic in internal/serverboot. INTEGIN_DEPLOY_MODE set directly in the
// environment takes precedence when --mode is left at its default.
//
// Schema migrations can be executed directly using the 'migrate' subcommand:
//
//	integin-server migrate [--rollback] [--db-url=<dsn>]
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"integin/internal/serverboot"
	"integin/migrations"
	"integin/pkg/agentmcp"
	"integin/pkg/domain"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "integin-server: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	// Support 'migrate' subcommand for container init & deployment pipelines
	if len(args) > 0 && args[0] == "migrate" {
		return runMigrate(args[1:])
	}

	fs := flag.NewFlagSet("integin-server", flag.ContinueOnError)
	mode := fs.String("mode", "", "deployment topology: cloud|sovereign|airgap (default: INTEGIN_DEPLOY_MODE or cloud)")
	migrateFlag := fs.Bool("migrate", false, "run database migrations before starting or exit")
	mcpFlag := fs.Bool("mcp", false, "run loopback MCP agent server over stdin/stdout")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if *mcpFlag {
		return runMCPLoopback()
	}

	if *migrateFlag {
		if err := runMigrate(nil); err != nil {
			return fmt.Errorf("startup migration: %w", err)
		}
	}

	if *mode != "" {
		if err := os.Setenv("INTEGIN_DEPLOY_MODE", *mode); err != nil {
			return fmt.Errorf("set INTEGIN_DEPLOY_MODE: %w", err)
		}
	}
	serverboot.Run()
	return nil
}

func runMigrate(args []string) error {
	fs := flag.NewFlagSet("migrate", flag.ContinueOnError)
	rollback := fs.Bool("rollback", false, "rollback the last applied migration")
	dbURLFlag := fs.String("db-url", "", "database connection string (overrides INTEGIN_MIGRATION_DB_URL / INTEGIN_DB_URL)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	dbURL := *dbURLFlag
	if dbURL == "" {
		dbURL = strings.TrimSpace(os.Getenv("INTEGIN_MIGRATION_DB_URL"))
	}
	if dbURL == "" {
		dbURL = strings.TrimSpace(os.Getenv("INTEGIN_DB_URL"))
	}
	if dbURL == "" {
		return fmt.Errorf("no database URL provided: set INTEGIN_MIGRATION_DB_URL, INTEGIN_DB_URL, or use --db-url")
	}

	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}

	if *rollback {
		reverted, err := migrations.RollbackLast(ctx, db)
		if err != nil {
			return fmt.Errorf("rollback failed: %w", err)
		}
		fmt.Printf("[+] Successfully rolled back: %s\n", reverted)
		return nil
	}

	fmt.Println("[*] Executing ledger-tracked schema migrations...")
	report, err := migrations.Migrate(ctx, db)
	if err != nil {
		return fmt.Errorf("migration run failed: %w", err)
	}

	fmt.Printf("[+] Migrations complete: %d discovered, %d already applied, %d newly applied in %v\n",
		report.TotalDiscovered, report.AlreadyApplied, report.NewlyApplied, report.Duration)
	for _, f := range report.AppliedFiles {
		fmt.Printf("    -> Applied: %s\n", f)
	}

	return nil
}

func runMCPLoopback() error {
	dispatcher := domain.NewDispatcher()
	domain.RegisterDefaultCommands(dispatcher)

	srv, err := agentmcp.NewServer(agentmcp.ServerConfig{
		ReadRoot:  ".",
		WriteRoot: ".",
	}, dispatcher)
	if err != nil {
		return fmt.Errorf("failed to create MCP server: %w", err)
	}

	ctx := context.Background()
	return srv.ServeStream(ctx, os.Stdin, os.Stdout)
}
