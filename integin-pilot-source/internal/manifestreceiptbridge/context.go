// Package manifestreceiptbridge validates pilot-only public receipt context.
package manifestreceiptbridge

import (
	"errors"
	"strings"
	"time"

	"integin/internal/manifestreceipts"
)

// Context is a validated, nonsecret run descriptor. It deliberately does not
// read process environment; a future pilot launcher must provide child-only
// values explicitly and must never expose them to the protected fixture.
type Context struct {
	RunID     string
	Directory string
	Version   string
}

// Load validates an explicit source-owned context. All-empty values mean the
// bridge is absent; partial values fail closed. This function never mounts a
// route, starts a candidate, or changes workflow authority.
func Load(runID, directory, version string, now func() time.Time) (*Context, *manifestreceipts.V2Writer, error) {
	if strings.TrimSpace(runID) == "" && strings.TrimSpace(directory) == "" && strings.TrimSpace(version) == "" {
		return nil, nil, nil
	}
	if strings.TrimSpace(runID) == "" || strings.TrimSpace(directory) == "" || strings.TrimSpace(version) == "" {
		return nil, nil, errors.New("manifest receipt bridge context is incomplete")
	}
	if version != "2" {
		return nil, nil, errors.New("manifest receipt bridge contract version is unsupported")
	}
	writer, err := manifestreceipts.NewV2Writer(runID, directory, now)
	if err != nil {
		return nil, nil, err
	}
	return &Context{RunID: runID, Directory: directory, Version: version}, writer, nil
}
