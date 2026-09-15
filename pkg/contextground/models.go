package contextground

import "time"

// GroundingRequest describes the validated context an AI prompt needs: which
// repository files and which AST symbols must be attached as ground truth.
type GroundingRequest struct {
	RepoRoot      string   `json:"repo_root"`
	FilePaths     []string `json:"file_paths"`
	Symbols       []string `json:"symbols"`
	MaxTokens     int      `json:"max_tokens"`
	RedactSecrets bool     `json:"redact_secrets"`
}

// GroundingPayload is the zero-hallucination, cryptographically attested
// context bundle handed to an AI prompt.
type GroundingPayload struct {
	Digest        string            `json:"digest"`
	Timestamp     time.Time         `json:"timestamp"`
	Files         []FileGrounding   `json:"files"`
	Symbols       []SymbolGrounding `json:"symbols"`
	TokenEstimate int               `json:"token_estimate"`
}

// FileGrounding carries one file's redacted content plus its SHA-256 so the
// consumer can independently re-attest what the prompt was shown.
type FileGrounding struct {
	Path      string `json:"path"`
	SHA256    string `json:"sha256"`
	Content   string `json:"content"`
	LineCount int    `json:"line_count"`
}

// SymbolGrounding pins one validated AST symbol (type, func, interface) to an
// exact file and line with its verbatim signature.
type SymbolGrounding struct {
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	File      string `json:"file"`
	Line      int    `json:"line"`
	Signature string `json:"signature"`
}
