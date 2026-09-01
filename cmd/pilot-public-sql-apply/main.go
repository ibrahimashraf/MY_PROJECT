// Command pilot-public-sql-apply applies a reviewed public fixture transaction
// using a caller-provided isolated DB URL. It never prints the URL or SQL body.
package main

import (
	"bytes"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/lib/pq"
)

func main() {
	sqlPath := flag.String("sql", "", "public fixture SQL transaction path")
	flag.Parse()
	if err := apply(*sqlPath); err != nil {
		fmt.Fprintln(os.Stdout, "PILOT_PUBLIC_SQL_APPLY_ERROR="+errorCode(err))
		os.Exit(1)
	}
	fmt.Println("PILOT_PUBLIC_SQL_APPLY_PASSED")
}

func errorCode(err error) string {
	switch err.Error() {
	case "public SQL path is required", "public SQL input is invalid", "public SQL transaction is invalid":
		return "PUBLIC_SQL_INVALID"
	case "isolated DB input is unavailable":
		return "DB_INPUT_UNAVAILABLE"
	case "isolated DB connection failed":
		return "DB_CONNECTION_FAILED"
	default:
		var databaseError *pq.Error
		if errors.As(err, &databaseError) {
			return "SQLSTATE_" + string(databaseError.Code)
		}
		return "UNCLASSIFIED"
	}
}

func apply(sqlPath string) error {
	if strings.TrimSpace(sqlPath) == "" {
		return errors.New("public SQL path is required")
	}
	info, err := os.Stat(sqlPath)
	if err != nil || !info.Mode().IsRegular() || info.Size() == 0 || info.Size() > 256<<10 {
		return errors.New("public SQL input is invalid")
	}
	//nolint:gosec // path is CLI arg/config for test tool
	statement, err := os.ReadFile(sqlPath)
	statement = bytes.TrimPrefix(statement, []byte{0xef, 0xbb, 0xbf})
	if err != nil || !strings.HasPrefix(strings.TrimSpace(string(statement)), "BEGIN;") || !strings.HasSuffix(strings.TrimSpace(string(statement)), "COMMIT;") {
		return errors.New("public SQL transaction is invalid")
	}
	dbURL := strings.TrimSpace(os.Getenv("INTEGIN_PILOT_MATRIX_DB_URL"))
	if dbURL == "" {
		return errors.New("isolated DB input is unavailable")
	}
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return errors.New("isolated DB connection failed")
	}
	defer db.Close()
	if _, err := db.Exec(string(statement)); err != nil {
		return err
	}
	return nil
}
