package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
)

// RSIRunner applies proposed patches and verifies them with go vet + go test,
// rolling back cleanly whenever a safety gate fails.
type RSIRunner struct {
	repoRoot string
}

// NewRunner returns a runner scoped to repoRoot. Target files and the package
// under verify are resolved relative to repoRoot.
func NewRunner(repoRoot string) *RSIRunner {
	return &RSIRunner{repoRoot: repoRoot}
}

// ApplyAndVerify enforces the mutation safety contract:
//  1. backup every targeted file,
//  2. apply the patch,
//  3. run go vet and go test -count=1 on the target package,
//  4. on any failure restore the backups; on success keep the changes,
//  5. always dispose of backups.
//
// Verification outcomes are returned in the IterationResult; only hard errors
// (invalid input, I/O, traversal attempts) are returned as errors.
func (r *RSIRunner) ApplyAndVerify(ctx context.Context, req MutationRequest) (*IterationResult, error) {
	if req.TestPackage == "" {
		return nil, fmt.Errorf("test_package is required")
	}
	if len(req.TargetFiles) == 0 {
		return nil, fmt.Errorf("target_files must not be empty")
	}
	if req.MaxAttempts < 1 {
		req.MaxAttempts = 1
	}

	start := time.Now()
	res := &IterationResult{Attempt: 1}
	defer func() { res.ExecutionDurationMs = time.Since(start).Milliseconds() }()

	backup, err := r.backupTargets(req.TargetFiles)
	if err != nil {
		return nil, fmt.Errorf("backup failed: %w", err)
	}

	keep := false
	defer func() {
		if keep {
			return
		}
		if rerr := restoreBackup(backup); rerr != nil {
			fmt.Fprintf(os.Stderr, "integin-rsi: rollback failed: %v\n", rerr)
		}
	}()

	if err := r.applyPatch(req.TargetFiles, req.Patch); err != nil {
		res.ErrorReason = "patch apply failed: " + err.Error()
		return res, nil
	}

	vetOut, err := r.runGo(ctx, "vet", req.TestPackage)
	res.VetOutput = vetOut
	if err != nil {
		res.ErrorReason = "go vet failed: " + summarize(vetOut)
		return res, nil
	}

	testOut, err := r.runGo(ctx, "test", "-count=1", req.TestPackage)
	res.TestOutput = testOut
	if err != nil {
		res.ErrorReason = "go test failed: " + summarize(testOut)
		return res, nil
	}

	res.Success = true
	keep = true
	return res, nil
}

type backupEntry struct {
	existed bool
	data    []byte
}

func (r *RSIRunner) backupTargets(targets []string) (map[string]backupEntry, error) {
	backup := make(map[string]backupEntry, len(targets))
	for _, t := range targets {
		abs, err := r.resolveTarget(t)
		if err != nil {
			return nil, err
		}
		data, err := os.ReadFile(abs)
		if err != nil {
			if os.IsNotExist(err) {
				backup[abs] = backupEntry{existed: false}
				continue
			}
			return nil, fmt.Errorf("backup %s: %w", t, err)
		}
		backup[abs] = backupEntry{existed: true, data: data}
	}
	return backup, nil
}

func restoreBackup(backup map[string]backupEntry) error {
	for abs, e := range backup {
		if e.existed {
			if err := os.WriteFile(abs, e.data, 0o644); err != nil {
				return fmt.Errorf("restore %s: %w", abs, err)
			}
			continue
		}
		if err := os.Remove(abs); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove %s: %w", abs, err)
		}
	}
	return nil
}

// resolveTarget maps a caller-supplied relative path to an absolute path that
// is guaranteed to stay inside the repository root.
func (r *RSIRunner) resolveTarget(rel string) (string, error) {
	if rel == "" {
		return "", fmt.Errorf("empty target file")
	}
	abs := filepath.Join(r.repoRoot, filepath.FromSlash(rel))
	rp, err := filepath.Rel(r.repoRoot, abs)
	if err != nil {
		return "", err
	}
	if rp == ".." || strings.HasPrefix(rp, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("target %q escapes repository root", rel)
	}
	return abs, nil
}

func (r *RSIRunner) runGo(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = r.repoRoot
	var out, errOut bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errOut
	err := cmd.Run()
	return strings.TrimSpace(out.String() + "\n" + errOut.String()), err
}

// --- unified diff parsing / application -----------------------------------

type hunk struct {
	path     string
	oldStart int
	newStart int
	oldCount int
	newCount int
	oldBlock []string
	newBlock []string
}

// applyPatch applies a unified-context diff restricted to reqTargets. Any
// hunk referencing a file outside the target set is a hard error.
func (r *RSIRunner) applyPatch(reqTargets []string, patch string) error {
	hunks, err := parsePatch(patch)
	if err != nil {
		return err
	}
	targetSet := make(map[string]bool, len(reqTargets))
	for _, t := range reqTargets {
		targetSet[normalizeRel(t)] = true
	}
	byFile := make(map[string][]hunk)
	for _, h := range hunks {
		rel := normalizeRel(h.path)
		if !targetSet[rel] {
			return fmt.Errorf("patch references %q, which is not in target_files", h.path)
		}
		byFile[rel] = append(byFile[rel], h)
	}
	for rel, hs := range byFile {
		if err := r.applyHunks(rel, hs); err != nil {
			return err
		}
	}
	return nil
}

func normalizeRel(p string) string {
	p = strings.TrimPrefix(p, "b/")
	p = strings.TrimPrefix(p, "a/")
	p = strings.TrimPrefix(p, "./")
	return filepath.ToSlash(p)
}

func (r *RSIRunner) applyHunks(rel string, hs []hunk) error {
	abs, err := r.resolveTarget(rel)
	if err != nil {
		return err
	}
	content, err := os.ReadFile(abs)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("read %s: %w", rel, err)
		}
		content = nil
	}

	var lines []string
	trailingNL := false
	switch {
	case len(content) == 0:
		lines = nil
	case bytes.HasSuffix(content, []byte("\n")):
		lines = strings.Split(string(content), "\n")
		lines = lines[:len(lines)-1]
		trailingNL = true
	default:
		lines = strings.Split(string(content), "\n")
	}

	shift := 0
	for _, h := range hs {
		base := h.oldStart - 1
		if base < 0 {
			if h.oldCount != 0 {
				return fmt.Errorf("hunk for %s has negative start", rel)
			}
			base = h.newStart - 1
		}
		idx := base + shift
		if idx < 0 || idx+len(h.oldBlock) > len(lines) {
			return fmt.Errorf("hunk for %s out of bounds", rel)
		}
		if !slices.Equal(lines[idx:idx+len(h.oldBlock)], h.oldBlock) {
			return fmt.Errorf("hunk for %s does not match file content at line %d", rel, base+1)
		}
		head := lines[:idx]
		tail := append([]string(nil), lines[idx+len(h.oldBlock):]...)
		merged := make([]string, 0, len(head)+len(h.newBlock)+len(tail))
		merged = append(merged, head...)
		merged = append(merged, h.newBlock...)
		merged = append(merged, tail...)
		lines = merged
		shift += len(h.newBlock) - len(h.oldBlock)
	}

	out := strings.Join(lines, "\n")
	if trailingNL || len(lines) > 0 {
		out += "\n"
	}
	return os.WriteFile(abs, []byte(out), 0o644)
}

func parsePatch(patch string) ([]hunk, error) {
	lines := strings.Split(patch, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	if len(lines) == 0 {
		return nil, fmt.Errorf("patch is empty")
	}

	path := ""
	var hunks []hunk
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		switch {
		case strings.HasPrefix(line, "+++ "):
			path = strings.TrimSpace(strings.TrimPrefix(line, "+++ "))
			path = normalizeRel(path)
		case strings.HasPrefix(line, "--- ") || strings.HasPrefix(line, "\\ "):
			// old path / "\ No newline at end of file" marker: skipped.
		case strings.HasPrefix(line, "@@ "):
			oldStart, newStart, oldCount, newCount, err := parseHunkHeader(line)
			if err != nil {
				return nil, err
			}
			h := hunk{
				path:     path,
				oldStart: oldStart,
				newStart: newStart,
				oldCount: oldCount,
				newCount: newCount,
			}
			j := i + 1
			for ; j < len(lines); j++ {
				l := lines[j]
				if strings.HasPrefix(l, "@@ ") || strings.HasPrefix(l, "+++ ") || strings.HasPrefix(l, "--- ") {
					break
				}
				switch {
				case l == "" || l[0] == ' ':
					body := l
					if l != "" {
						body = l[1:]
					}
					h.oldBlock = append(h.oldBlock, body)
					h.newBlock = append(h.newBlock, body)
				case l[0] == '-':
					h.oldBlock = append(h.oldBlock, l[1:])
				case l[0] == '+':
					h.newBlock = append(h.newBlock, l[1:])
				case l[0] == '\\':
					// no-newline marker: treated as cosmetic.
				default:
					return nil, fmt.Errorf("malformed hunk line %q", l)
				}
			}
			if h.oldCount != len(h.oldBlock) || h.newCount != len(h.newBlock) {
				return nil, fmt.Errorf("hunk header count mismatch (old %d lines, want %d; new %d lines, want %d)", len(h.oldBlock), h.oldCount, len(h.newBlock), h.newCount)
			}
			if h.path == "" {
				return nil, fmt.Errorf("hunk on line %d has no target file", i+1)
			}
			i = j - 1
			hunks = append(hunks, h)
		}
	}
	if len(hunks) == 0 {
		return nil, fmt.Errorf("no hunks found in patch")
	}
	return hunks, nil
}

// parseHunkHeader parses "@@ -l[,c] +l[,c] @@" (count defaults to 1).
func parseHunkHeader(line string) (oldStart, newStart, oldCount, newCount int, err error) {
	fields := strings.Fields(line)
	if len(fields) < 4 || fields[0] != "@@" || !strings.HasPrefix(fields[1], "-") || !strings.HasPrefix(fields[2], "+") {
		return 0, 0, 0, 0, fmt.Errorf("malformed hunk header %q", line)
	}
	parseRange := func(s string) (start, count int, err error) {
		s = s[1:]
		if s == "" {
			return 0, 0, fmt.Errorf("empty range")
		}
		parts := strings.Split(s, ",")
		if len(parts) > 2 {
			return 0, 0, fmt.Errorf("bad range")
		}
		start, err = strconv.Atoi(parts[0])
		if err != nil {
			return 0, 0, err
		}
		count = 1
		if len(parts) == 2 {
			count, err = strconv.Atoi(parts[1])
			if err != nil {
				return 0, 0, err
			}
		}
		return start, count, nil
	}
	oldStart, oldCount, err = parseRange(fields[1])
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("malformed old range in %q", line)
	}
	newStart, newCount, err = parseRange(fields[2])
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("malformed new range in %q", line)
	}
	return oldStart, newStart, oldCount, newCount, nil
}

func summarize(out string) string {
	var lines []string
	for _, l := range strings.Split(out, "\n") {
		if l = strings.TrimSpace(l); l != "" {
			lines = append(lines, l)
		}
	}
	if len(lines) == 0 {
		return "no output"
	}
	if len(lines) > 3 {
		lines = lines[:3]
	}
	msg := strings.Join(lines, "; ")
	if len(msg) > 400 {
		msg = msg[:400] + "..."
	}
	return msg
}
