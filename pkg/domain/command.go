package domain

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"
)

var (
	ErrUnknownCommand  = errors.New("unknown command id")
	ErrInvalidParams   = errors.New("invalid command parameters")
	ErrTenantRequired  = errors.New("tenant_id is required for command execution")
	ErrCommandDisabled = errors.New("command is currently disabled")
)

// Command represents a single atomic, serializable mutation request (inspired by PhotoCraft's Session::execute).
// Every user action, batch CLI call, or AI agent (via MCP) executes through this unified schema.
type Command struct {
	ID             string          `json:"id"`              // e.g. "passport.custody.transfer"
	TenantID       string          `json:"tenant_id"`       // Bound to PostgreSQL RLS context
	OrganizationID string          `json:"organization_id"` // Bound to PostgreSQL RLS context
	ActorID        string          `json:"actor_id"`        // User, Service Account, or Agent DID
	Payload        json.RawMessage `json:"payload"`         // JSON parameters validated by target handler
	Timestamp      time.Time       `json:"timestamp"`
}

// CommandResult is the deterministic response returned after executing a Command.
type CommandResult struct {
	CommandID string          `json:"command_id"`
	Status    string          `json:"status"` // "SUCCESS" or "FAILED"
	Revision  uint64          `json:"revision"`
	Output    json.RawMessage `json:"output,omitempty"`
	Error     string          `json:"error,omitempty"`
}

// Handler defines the execution contract for a specific registered command ID.
type Handler func(ctx context.Context, cmd Command) (json.RawMessage, error)

// Dispatcher manages the command registry and enforces RLS context propagation.
type Dispatcher struct {
	mu       sync.RWMutex
	handlers map[string]Handler
	revision uint64
}

// NewDispatcher initializes an empty command dispatcher.
func NewDispatcher() *Dispatcher {
	return &Dispatcher{
		handlers: make(map[string]Handler),
	}
}

// Register maps a stable command ID string to a domain Handler.
func (d *Dispatcher) Register(commandID string, h Handler) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.handlers[commandID] = h
}

// Execute routes a command to its handler while validating tenant boundaries and incrementing state revision.
func (d *Dispatcher) Execute(ctx context.Context, cmd Command) (CommandResult, error) {
	if cmd.TenantID == "" {
		return CommandResult{
			CommandID: cmd.ID,
			Status:    "FAILED",
			Error:     ErrTenantRequired.Error(),
		}, ErrTenantRequired
	}

	d.mu.RLock()
	handler, exists := d.handlers[cmd.ID]
	d.mu.RUnlock()

	if !exists {
		err := fmt.Errorf("%w: %s", ErrUnknownCommand, cmd.ID)
		return CommandResult{
			CommandID: cmd.ID,
			Status:    "FAILED",
			Error:     err.Error(),
		}, err
	}

	d.mu.Lock()
	d.revision++
	currentRev := d.revision
	d.mu.Unlock()

	out, err := handler(ctx, cmd)
	if err != nil {
		return CommandResult{
			CommandID: cmd.ID,
			Status:    "FAILED",
			Revision:  currentRev,
			Error:     err.Error(),
		}, err
	}

	return CommandResult{
		CommandID: cmd.ID,
		Status:    "SUCCESS",
		Revision:  currentRev,
		Output:    out,
	}, nil
}
