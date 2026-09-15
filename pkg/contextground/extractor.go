package contextground

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

// maxFileBytes guards against pulling a monster file into a prompt pool.
const maxFileBytes = 8 << 20 // 8 MiB

// charsPerToken is the cost model used for TokenEstimate: roughly one token
// per 4 chars. Deterministic, offline, dependency-free.
const charsPerToken = 4

// secretPatterns are standard high-signal secret shapes. Replacement happens
// before content is stored, hashed, or transmitted.
var secretPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?is)-----BEGIN [A-Z ]*PRIVATE KEY-----.*?-----END [A-Z ]*PRIVATE KEY-----`),
	regexp.MustCompile(`\bAIza[0-9A-Za-z_-]{35}\b`),
	regexp.MustCompile(`(?i)\bbearer\s+[0-9a-zA-Z_.~-]{16,}\b`),
	regexp.MustCompile(`\bAKIA[0-9A-Z]{16}\b`),
	regexp.MustCompile(`(?i)\bsk-[a-z0-9_]{20,}\b`),
}

// ContextGrounder assembles validated, attested context for prompt grounding.
// It is stateless and safe for concurrent use.
type ContextGrounder struct{}

// New returns a ContextGrounder.
func New() *ContextGrounder {
	return &ContextGrounder{}
}

// Ground resolves requested files and AST symbols under RepoRoot, redacts
// secrets, computes a deterministic SHA-256 over the included file contents,
// and returns the attested payload. FilePaths and Symbols must not both be
// empty. FilePaths are resolved relative to RepoRoot (defaults to ".") and
// may not escape it.
func (g *ContextGrounder) Ground(ctx context.Context, req GroundingRequest) (*GroundingPayload, error) {
	if len(req.FilePaths) == 0 && len(req.Symbols) == 0 {
		return nil, errors.New("grounding requires at least one file_path or symbol")
	}

	root := req.RepoRoot
	if root == "" {
		root = "."
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve repo root: %w", err)
	}

	// Read every requested file once; cache raw source only when we must
	// parse it for symbol extraction.
	paths, err := g.resolveFiles(ctx, absRoot, req.FilePaths, len(req.Symbols) > 0)
	if err != nil {
		return nil, err
	}

	grounds := make([]FileGrounding, 0, len(paths))
	parsed := make(map[string]string, len(paths)) // rel path -> raw source
	for _, p := range paths {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		rel, rerr := filepath.Rel(absRoot, p)
		if rerr != nil {
			return nil, fmt.Errorf("relative path: %w", rerr)
		}
		fg, raw, rerr := readGround(p, rel, req.RedactSecrets)
		if rerr != nil {
			return nil, rerr
		}
		grounds = append(grounds, fg)
		if strings.HasSuffix(strings.ToLower(rel), ".go") {
			parsed[rel] = raw
		}
	}
	sort.Slice(grounds, func(i, j int) bool { return grounds[i].Path < grounds[j].Path })

	var symbols []SymbolGrounding
	if len(req.Symbols) > 0 {
		symbols, err = extractSymbols(ctx, grounds, parsed, req.Symbols)
		if err != nil {
			return nil, err
		}
	}

	if req.MaxTokens > 0 {
		grounds = applyTokenBudget(grounds, req.MaxTokens)
	}
	symbols = keepIncludedFiles(symbols, grounds)

	digest, estimate := digestAndEstimate(grounds)

	return &GroundingPayload{
		Digest:        digest,
		Timestamp:     time.Now().UTC(),
		Files:         grounds,
		Symbols:       symbols,
		TokenEstimate: estimate,
	}, nil
}

// resolveFiles returns the deduplicated, sorted absolute paths to read. When
// paths is empty (symbol-only request), every *.go file under root is
// discovered via a cancelled-context-aware walk.
func (g *ContextGrounder) resolveFiles(ctx context.Context, root string, paths []string, discover bool) ([]string, error) {
	seen := make(map[string]struct{})
	resolved := make([]string, 0, len(paths))
	for _, fp := range paths {
		if fp == "" {
			continue
		}
		full, err := safeJoin(root, fp)
		if err != nil {
			return nil, err
		}
		if _, ok := seen[full]; ok {
			continue
		}
		seen[full] = struct{}{}
		resolved = append(resolved, full)
	}
	if len(resolved) == 0 && discover {
		var walked []string
		err := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if err := ctx.Err(); err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			if strings.HasSuffix(strings.ToLower(d.Name()), ".go") {
				walked = append(walked, p)
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("discover go files: %w", err)
		}
		resolved = walked
	}
	sort.Strings(resolved)
	return resolved, nil
}

// safeJoin joins fp onto root after rejecting absolute paths and any traversal
// attempt, and verifies the result stays inside root via a Rel boundary check.
func safeJoin(root, fp string) (string, error) {
	if filepath.IsAbs(fp) {
		return "", fmt.Errorf("absolute paths are not allowed: %s", fp)
	}
	for _, comp := range strings.Split(filepath.ToSlash(filepath.Clean(fp)), "/") {
		if comp == ".." {
			return "", errors.New("path traversal rejected: " + fp)
		}
	}
	full := filepath.Clean(filepath.Join(root, fp))
	rel, err := filepath.Rel(root, full)
	if err != nil {
		return "", fmt.Errorf("resolve %s: %w", fp, err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", errors.New("path escapes repository root: " + fp)
	}
	return full, nil
}

// readGround reads one file, redacts secrets, and returns its grounding entry
// plus the raw source (needed for verbatim signature extraction) when the
// caller will parse it.
func readGround(abs, rel string, redact bool) (FileGrounding, string, error) {
	f, err := os.Open(abs)
	if err != nil {
		return FileGrounding{}, "", fmt.Errorf("open %s: %w", rel, err)
	}
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return FileGrounding{}, "", fmt.Errorf("stat %s: %w", rel, err)
	}
	if info.Size() > maxFileBytes {
		f.Close()
		return FileGrounding{}, "", fmt.Errorf("file exceeds %d byte limit: %s", maxFileBytes, rel)
	}
	raw, err := io.ReadAll(f)
	f.Close()
	if err != nil {
		return FileGrounding{}, "", fmt.Errorf("read %s: %w", rel, err)
	}
	content := string(raw)
	if redact {
		content = redactSecrets(content)
	}
	return FileGrounding{
		Path:      rel,
		SHA256:    sha256Hex(content),
		Content:   content,
		LineCount: countLines(content),
	}, string(raw), nil
}

// extractSymbols finds every requested symbol by its AST declaration across
// the parsed files. Results are returned de-duplicated in request order, and
// the first declaring file wins.
func extractSymbols(ctx context.Context, grounds []FileGrounding, parsed map[string]string, requested []string) ([]SymbolGrounding, error) {
	names := make([]string, 0, len(requested))
	found := make(map[string]*SymbolGrounding, len(requested))
	{
		seen := make(map[string]struct{}, len(requested))
		for _, raw := range requested {
			name := strings.TrimSpace(raw)
			if name == "" {
				continue
			}
			if _, ok := seen[name]; ok {
				continue
			}
			seen[name] = struct{}{}
			names = append(names, name)
			found[name] = nil
		}
	}

	for _, fg := range grounds {
		raw, ok := parsed[fg.Path]
		if !ok {
			continue
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		fset := token.NewFileSet()
		node, err := parser.ParseFile(fset, fg.Path, raw, 0)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", fg.Path, err)
		}
		ast.Inspect(node, func(n ast.Node) bool {
			switch d := n.(type) {
			case *ast.FuncDecl:
				if found[d.Name.Name] == nil {
					pos := fset.Position(d.Pos())
					found[d.Name.Name] = &SymbolGrounding{
						Name:      d.Name.Name,
						Kind:      "func",
						File:      fg.Path,
						Line:      pos.Line,
						Signature: sliceSignature(raw, fset, d.Pos(), d.Body.Pos()),
					}
				}
			case *ast.GenDecl:
				for _, spec := range d.Specs {
					ts, ok := spec.(*ast.TypeSpec)
					if !ok || found[ts.Name.Name] != nil {
						continue
					}
					kind := "type"
					if _, isIface := ts.Type.(*ast.InterfaceType); isIface {
						kind = "interface"
					}
					pos := fset.Position(ts.Pos())
					found[ts.Name.Name] = &SymbolGrounding{
						Name:      ts.Name.Name,
						Kind:      kind,
						File:      fg.Path,
						Line:      pos.Line,
						Signature: sliceSignature(raw, fset, d.Pos(), ts.Type.End()),
					}
				}
			}
			return true
		})
	}

	out := make([]SymbolGrounding, 0, len(names))
	for _, name := range names {
		if sg := found[name]; sg != nil {
			out = append(out, *sg)
		}
	}
	return out, nil
}

// sliceSignature extracts the verbatim source text between two AST positions,
// e.g. "func (c *C) Get() error" or "type T struct {...}", trimmed of
// surrounding whitespace and redacted in case a signature carries a literal.
func sliceSignature(src string, fset *token.FileSet, start, end token.Pos) string {
	s := fset.Position(start).Offset
	e := fset.Position(end).Offset
	if s < 0 || e < s || e >= len(src) {
		return ""
	}
	return redactSecrets(strings.TrimSpace(src[s:e]))
}

// applyTokenBudget drops trailing files once the estimated token budget is
// spent, trimming the file that would overflow. Deterministic: files are
// already sorted by path.
func applyTokenBudget(grounds []FileGrounding, budget int) []FileGrounding {
	kept := make([]FileGrounding, 0, len(grounds))
	for _, fg := range grounds {
		est := estimateTokens(fg.Content)
		if est <= budget {
			kept = append(kept, fg)
			budget -= est
			continue
		}
		if maxChars := budget * charsPerToken; maxChars > 0 {
			fg.Content = truncateToValid(fg.Content, maxChars)
			fg.LineCount = countLines(fg.Content)
			fg.SHA256 = sha256Hex(fg.Content)
			kept = append(kept, fg)
		}
		break
	}
	return kept
}

// truncateToValid clips content to maxChars bytes, backing off to the nearest
// valid UTF-8 boundary so the payload never contains a truncated rune.
func truncateToValid(s string, maxChars int) string {
	if len(s) <= maxChars {
		return s
	}
	b := s[:maxChars]
	for maxChars > 0 && !utf8.ValidString(b) {
		maxChars--
		b = s[:maxChars]
	}
	return b
}

// digestAndEstimate hashes the concatenated sorted file contents and sums
// their token estimates. Both are fully deterministic.
func digestAndEstimate(grounds []FileGrounding) (string, int) {
	h := sha256.New()
	total := 0
	for _, fg := range grounds {
		h.Write([]byte(fg.Content))
		total += estimateTokens(fg.Content)
	}
	return hex.EncodeToString(h.Sum(nil)), total
}

// keepIncludedFiles drops symbol records whose file was cut by the token
// budget, so every symbol points at a file actually present in the payload.
func keepIncludedFiles(symbols []SymbolGrounding, grounds []FileGrounding) []SymbolGrounding {
	included := make(map[string]struct{}, len(grounds))
	for _, fg := range grounds {
		included[fg.Path] = struct{}{}
	}
	kept := symbols[:0]
	for _, sg := range symbols {
		if _, ok := included[sg.File]; ok {
			kept = append(kept, sg)
		}
	}
	return kept
}

// estimateTokens models tokens as chars/charsPerToken.
func estimateTokens(s string) int {
	if s == "" {
		return 0
	}
	return len(s)/charsPerToken + 1
}

func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func countLines(s string) int {
	if s == "" {
		return 0
	}
	n := strings.Count(s, "\n")
	if !strings.HasSuffix(s, "\n") {
		n++
	}
	return n
}

// redactSecrets replaces known-high-signal secret shapes with [REDACTED].
func redactSecrets(s string) string {
	for _, re := range secretPatterns {
		s = re.ReplaceAllString(s, "[REDACTED]")
	}
	return s
}
