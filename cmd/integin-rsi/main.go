package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

const usageText = `integin-rsi — closed-loop mutation verification runner.

Usage:
  integin-rsi [flags]
  integin-rsi < mutation_request.json

Flags:
  --target <file>     repository-relative file the patch may touch (repeatable)
  --package <pkg>     Go package to verify with go vet and go test -count=1
  --help              show this help

With no flags, a MutationRequest JSON document is read from stdin, e.g.:
  {"target_files":["pkg/x/a.go"],"patch":"--- a/pkg/x/a.go\n+++ b/pkg/x/a.go\n@@ -1,1 +1,1 @@\n-old\n+new\n","test_package":"./pkg/x"}

On success (vet + tests green) the IterationResult is printed and the process
exits 0. On verification failure it exits 1; on usage/input error it exits 2.
`

type targetList []string

func (t *targetList) String() string { return strings.Join(*t, ",") }
func (t *targetList) Set(v string) error {
	*t = append(*t, v)
	return nil
}

func main() {
	fs := flag.NewFlagSet("integin-rsi", flag.ExitOnError)
	fs.Usage = func() { fmt.Fprint(os.Stderr, usageText) }
	help := fs.Bool("help", false, "show help")
	var targets targetList
	fs.Var(&targets, "target", "repository-relative file the patch may touch (repeatable)")
	pkg := fs.String("package", "", "Go package to verify")
	if err := fs.Parse(os.Args[1:]); err != nil {
		os.Exit(2)
	}
	if *help {
		fs.Usage()
		os.Exit(0)
	}

	req, err := buildRequest(targets, *pkg, os.Stdin)
	if err != nil {
		die(err.Error())
	}

	cwd, err := os.Getwd()
	if err != nil {
		die("get working directory: " + err.Error())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	res, err := NewRunner(cwd).ApplyAndVerify(ctx, *req)
	if err != nil {
		die(err.Error())
	}
	out, err := json.Marshal(res)
	if err != nil {
		die("encode result: " + err.Error())
	}
	fmt.Println(string(out))
	if !res.Success {
		os.Exit(1)
	}
}

// buildRequest prefers explicit flags; otherwise it decodes a MutationRequest
// from the supplied reader (os.Stdin for the CLI).
func buildRequest(targets targetList, pkg string, stdin io.Reader) (*MutationRequest, error) {
	if len(targets) > 0 || pkg != "" {
		if len(targets) == 0 {
			return nil, fmt.Errorf("--target is required when --package is set")
		}
		if pkg == "" {
			return nil, fmt.Errorf("--package is required when --target is set")
		}
		return &MutationRequest{TargetFiles: targets, TestPackage: pkg, MaxAttempts: 1}, nil
	}
	data, err := io.ReadAll(stdin)
	if err != nil {
		return nil, fmt.Errorf("read stdin: %w", err)
	}
	var req MutationRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("decode MutationRequest JSON: %w", err)
	}
	return &req, nil
}

func die(msg string) {
	fmt.Fprintln(os.Stderr, "integin-rsi:", msg)
	os.Exit(2)
}
