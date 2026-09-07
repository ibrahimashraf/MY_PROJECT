// Command standards-compiler ingests citation card files (YAML or JSON),
// validates them against the standards citation contract and the Safe Harbor
// store-URL guard, indexes them, and writes the air-gapped offline cache that
// a shipboard device consumes for abstract standards search and manifest
// binding. It is fully offline: no network calls are made, and no raw standard
// text is ever stored.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"integin/pkg/rulesengine"
	"integin/pkg/standardsync"
	"integin/pkg/standardsync/connectors"

	"cel.dev/cel-go/cel"
	goccyyaml "github.com/goccy/go-yaml"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "list-bodies":
		listBodies()
	case "compile":
		compile(os.Args[2:])
	case "verify":
		verify(os.Args[2:])
	case "help", "-h", "--help":
		usage()
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `standards-compiler: ingest, validate, index, and cache standards citation cards.

Commands:
  list-bodies                         print the twelve supported standard bodies
  compile -in file[,file...] -out cache.json [--keep-payload-mode]
      Parse cards from YAML or JSON files, run citation-contract + Safe Harbor
      validation, index them, and write the offline cache.
  verify -cache cache.json [-query "mobile crane boom"]
      Check cache integrity and optionally run an abstract search against it.

Cards are citation metadata only. Payload extraction of any kind is refused.`)
}

func listBodies() {
	for _, body := range connectors.All() {
		fmt.Printf("%-10s %-8s %s\n  official: %s\n  hosts:    %s\n", body.Slug(), body.Country(), body.Name(), body.StoreBaseURL(), strings.Join(body.StoreHosts(), ", "))
	}
}

func compile(args []string) {
	fs := flag.NewFlagSet("compile", flag.ExitOnError)
	inFiles := fs.String("in", "", "comma-separated card files (.yaml or .json)")
	outFile := fs.String("out", "", "output cache path (.json)")
	emitCEL := fs.Bool("emit-cellist", false, "compile each ACTIVE card into a validated Google CEL gate expression")
	_ = fs.String("keep-payload-mode", "", "compat no-op; payloads are always refused")
	if err := fs.Parse(args); err != nil {
		die(err)
	}
	if *inFiles == "" || *outFile == "" {
		die(errors.New("-in and -out are required"))
	}

	cards, err := loadCards(*inFiles)
	if err != nil {
		die(fmt.Errorf("load cards: %w", err))
	}
	if len(cards) == 0 {
		die(errors.New("no cards loaded"))
	}

	// Resolver performs full lifecycle validation across the whole universe,
	// so a lone orphan superseded card or a duplicate DID aborts here.
	if _, err := standardsync.NewResolver(cards); err != nil {
		die(fmt.Errorf("lifecycle validation: %w", err))
	}
	idx := standardsync.NewAbstractIndex(cards)

	cache, err := standardsync.BuildLocalCacheFromCards(idx.Cards())
	if err != nil {
		die(err)
	}
	if err := cache.Store(*outFile, idx.Cards(), time.Now()); err != nil {
		die(fmt.Errorf("write cache: %w", err))
	}

	fmt.Printf("compiled %d cards -> %s\n", len(idx.Cards()), *outFile)
	fmt.Printf("  bodies:      %s\n", strings.Join(bodiesInCards(cards), ", "))
	fmt.Printf("  index:       %d cards, %d-dim vectors\n", idx.Len(), 128)
	fmt.Printf("  bindings:    %d ACTIVE\n", countActive(cards))
	if *emitCEL {
		fmt.Println(celList(idx.Cards()))
	}
}

// celVars declares the two variables every emitted standard gate uses.
func celVars() map[string]*cel.Type {
	return map[string]*cel.Type{
		"standard_did": cel.StringType,
		"lifecycle":    cel.StringType,
	}
}

// celList compiles the ACTIVE-revision gate of every card through the D2.1
// rulesengine sandbox, proving the ingest->CEL pipeline end to end, and
// returns the validated expressions one per line.
func celList(cards []standardsync.StandardMetadataCard) string {
	evaluator, err := rulesengine.NewEvaluator()
	if err != nil {
		die(fmt.Errorf("rulesengine evaluator: %w", err))
	}
	var b strings.Builder
	b.WriteString("validated CEL gates (AST-compiled, sandbox-checked):\n")
	for _, card := range cards {
		if card.LifecycleState != standardsync.LifecycleActive {
			continue
		}
		expr := fmt.Sprintf(`standard_did == %q && lifecycle == "ACTIVE"`, card.StandardDID)
		program, err := evaluator.Compile("standard-binding-"+card.StandardDID, expr, celVars())
		if err != nil {
			die(fmt.Errorf("CEL compile %s: %w", card.StandardDID, err))
		}
		vars := rulesengine.NewVarSet("standard_did", "lifecycle")
		vars.Put("standard_did", card.StandardDID)
		vars.Put("lifecycle", string(card.LifecycleState))
		pass, err := program.Evaluate(context.Background(), vars)
		if err != nil || !pass {
			die(fmt.Errorf("CEL evaluate %s: pass=%v err=%v", card.StandardDID, pass, err))
		}
		fmt.Fprintf(&b, "  %s  ==  %s\n", card.StandardDID, expr)
	}
	return b.String()
}

func verify(args []string) {
	fs := flag.NewFlagSet("verify", flag.ExitOnError)
	cachePath := fs.String("cache", "", "cache file to open")
	query := fs.String("query", "", "optional abstract search query")
	if err := fs.Parse(args); err != nil {
		die(err)
	}
	if *cachePath == "" {
		die(errors.New("-cache is required"))
	}
	cache, err := standardsync.OpenLocalCache(*cachePath)
	if err != nil {
		die(fmt.Errorf("open cache: %w", err))
	}
	resolver, err := standardsync.NewResolver(cache.Cards())
	if err != nil {
		die(fmt.Errorf("cache cards violate lifecycle rules: %w", err))
	}
	fmt.Printf("cache OK: %d cards, schema v1, checksum verified\n", len(cache.Cards()))
	if *query != "" {
		engine := standardsync.NewEngine(resolver, cache.Index())
		result, err := engine.Search(*query)
		if err != nil {
			die(err)
		}
		fmt.Printf("\nquery: %s (%s, %d cards indexed)\n", result.Query, result.Elapsed.Round(time.Microsecond), result.Metadata.CardsIndexed)
		fmt.Printf("summary: %s\n\n", result.QuerySummary)
		for _, m := range result.PrimaryMatches {
			fmt.Printf("  [primary]   %-24s %-10s %s\n", m.DID, m.State, m.Title)
		}
		for _, m := range result.DeprecatedMatches {
			fmt.Printf("  [deprecated] %-24s %-10s %s\n", m.DID, m.State, m.Title)
		}
	}
}

// loadCards parses every listed file as YAML or JSON card documents.
func loadCards(files string) ([]standardsync.StandardMetadataCard, error) {
	var cards []standardsync.StandardMetadataCard
	for _, file := range strings.Split(files, ",") {
		file = strings.TrimSpace(file)
		if file == "" {
			continue
		}
		raw, err := os.ReadFile(file)
		if err != nil {
			return nil, err
		}
		ext := strings.ToLower(filepath.Ext(file))
		if ext == ".json" {
			var list []standardsync.StandardMetadataCard
			if err := json.Unmarshal(raw, &list); err != nil {
				var single standardsync.StandardMetadataCard
				if singleErr := json.Unmarshal(raw, &single); singleErr != nil {
					return nil, fmt.Errorf("%s: %w (want JSON list or single card)", file, err)
				}
				cards = append(cards, single)
				continue
			}
			cards = append(cards, list...)
			continue
		}
		if ext == ".yaml" || ext == ".yml" {
			var list []standardsync.StandardMetadataCard
			if err := goccyyaml.Unmarshal(raw, &list); err != nil {
				var single standardsync.StandardMetadataCard
				if singleErr := goccyyaml.Unmarshal(raw, &single); singleErr != nil {
					return nil, fmt.Errorf("%s: %w (want YAML list or single card)", file, err)
				}
				cards = append(cards, single)
				continue
			}
			cards = append(cards, list...)
			continue
		}
		return nil, fmt.Errorf("%s: unsupported extension %q (use .yaml/.yml/.json)", file, ext)
	}
	return cards, nil
}

// bodiesInCards lists the distinct body slugs among loaded cards.
func bodiesInCards(cards []standardsync.StandardMetadataCard) []string {
	seen := map[string]bool{}
	var out []string
	for _, card := range cards {
		if !seen[card.StandardBody] {
			seen[card.StandardBody] = true
			out = append(out, card.StandardBody)
		}
	}
	return out
}

// countActive tallies cards in the ACTIVE lifecycle state.
func countActive(cards []standardsync.StandardMetadataCard) int {
	n := 0
	for _, card := range cards {
		if card.LifecycleState == standardsync.LifecycleActive {
			n++
		}
	}
	return n
}

func die(err error) {
	fmt.Fprintln(os.Stderr, "standards-compiler:", err)
	os.Exit(1)
}
