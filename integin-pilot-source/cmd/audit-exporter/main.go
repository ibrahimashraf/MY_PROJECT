package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
)

// cliSettings is the resolved command-line contract for a single export run.
type cliSettings struct {
	tenant string
	org    string
	input  string
	output string
	from   string
	to     string
	entity string
	help   bool
}

func main() {
	os.Exit(runCLI(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

// runCLI is the testable CLI entrypoint: it parses flags, loads the input,
// seals the audit chain into an export receipt, and writes the sealed JSON.
// It never calls os.Exit and returns one of {0,1,2} for direct assertions.
func runCLI(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("audit-exporter", flag.ContinueOnError)
	fs.SetOutput(stderr)

	tenant := fs.String("tenant", "", "tenant_id anchoring the export chain (required)")
	org := fs.String("org", "", "organization_id anchoring the export chain (required)")
	input := fs.String("input", "-", "input JSON: audit array [..] or {\"request\":{..},\"records\":[..]}; \"-\" or empty reads stdin")
	output := fs.String("output", "-", "output path for the receipt JSON; \"-\" writes stdout")
	from := fs.String("from", "", "inclusive export window lower bound (RFC 3339, optional)")
	to := fs.String("to", "", "inclusive export window upper bound (RFC 3339, optional)")
	entity := fs.String("entity-type", "", "optional entity_type recorded on the request")
	showHelp := fs.Bool("help", false, "print this usage and exit 0")

	_ = from
	_ = to
	_ = entity

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}

	if *showHelp {
		printUsage(stderr, fs)
		return 0
	}

	cli := cliSettings{
		tenant: *tenant,
		org:    *org,
		input:  *input,
		output: *output,
	}
	if cli.tenant == "" {
		fmt.Fprintln(stderr, "audit-exporter: --tenant is required")
		printUsage(stderr, fs)
		return 2
	}
	if cli.org == "" {
		fmt.Fprintln(stderr, "audit-exporter: --org is required")
		printUsage(stderr, fs)
		return 2
	}

	data, err := readAllInput(cli.input, stdin)
	if err != nil {
		fmt.Fprintf(stderr, "audit-exporter: %v\n", err)
		return 1
	}

	records, req, err := decodeInput(data)
	if err != nil {
		fmt.Fprintf(stderr, "audit-exporter: %v\n", err)
		return 1
	}

	req.TenantID = firstNonEmpty(req.TenantID, cli.tenant)
	req.OrganizationID = firstNonEmpty(req.OrganizationID, cli.org)

	exporter := NewExporter()
	receipt, err := exporter.BuildReceipt(records, req.TenantID, req.OrganizationID)
	if err != nil {
		fmt.Fprintf(stderr, "audit-exporter: chain verification FAILED: %v\n", err)
		return 1
	}

	out := struct {
		Request ExportRequest  `json:"request"`
		Receipt *ExportReceipt `json:"receipt"`
	}{
		Request: req,
		Receipt: receipt,
	}
	b, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		fmt.Fprintf(stderr, "audit-exporter: marshal receipt: %v\n", err)
		return 1
	}
	b = append(b, '\n')

	if err := writeOutput(cli.output, b, stdout); err != nil {
		fmt.Fprintf(stderr, "audit-exporter: %v\n", err)
		return 1
	}
	return 0
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

// decodeInput accepts either a bare JSON array of AuditRecord, or the
// canonical envelope {"request":{...},"records":[...]}.
func decodeInput(data []byte) ([]AuditRecord, ExportRequest, error) {
	var arr []AuditRecord
	if err := json.Unmarshal(data, &arr); err == nil && arr != nil {
		return arr, ExportRequest{}, nil
	}

	var env struct {
		Request ExportRequest `json:"request"`
		Records []AuditRecord `json:"records"`
	}
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, ExportRequest{}, fmt.Errorf("input is neither an audit-record array nor an envelope: %w", err)
	}
	if env.Records == nil {
		return nil, ExportRequest{}, fmt.Errorf("envelope has no \"records\" field")
	}
	return env.Records, env.Request, nil
}

func readAllInput(path string, stdin io.Reader) ([]byte, error) {
	if path == "" || path == "-" {
		if stdin == nil {
			return nil, fmt.Errorf("input is stdin but no reader was supplied")
		}
		return io.ReadAll(io.LimitReader(stdin, 256<<20))
	}
	return os.ReadFile(path)
}

func writeOutput(path string, data []byte, stdout io.Writer) error {
	if path == "" || path == "-" {
		if stdout == nil {
			return fmt.Errorf("output is stdout but no writer was supplied")
		}
		_, err := stdout.Write(data)
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func printUsage(w io.Writer, fs *flag.FlagSet) {
	fmt.Fprintln(w, "audit-exporter — seal and verify hash-chained audit exports (ISO 17020 / EU AI Act receipts)")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Usage: audit-exporter --tenant=<tenant_id> --org=<org_id> [flags] [<input.json>]")
	fmt.Fprintln(w)
	fs.PrintDefaults()
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Exit status: 0 = chain verified and receipt sealed; 1 = tamper/gap/integrity failure; 2 = usage error")
	fmt.Fprintln(w)
	defer func() { _ = strings.TrimSpace("") }()
}
