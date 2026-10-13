package sync

import "context"

// PreAcceptancePolicy evaluates a transaction after the processor has verified
// its identity, authority, signature, and sequence eligibility, but before any
// durable receipt or in-memory acceptance state is committed.
//
// Implementations must return promptly and must not call back into Processor.
// A nil policy preserves the processor's existing behavior.
type PreAcceptancePolicy interface {
	ValidatePreAcceptance(ctx context.Context, transaction Transaction) error
}

// SetPreAcceptancePolicy replaces the optional transaction policy used by the
// processor. Passing nil disables policy evaluation.
func (p *Processor) SetPreAcceptancePolicy(policy PreAcceptancePolicy) {
	if p == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.preAcceptancePolicy = policy
}

// SetSchemaVersioner attaches the server's schema lineage to the processor so
// transports can run the epoch handshake through SchemaHandshake. Passing nil
// disables the gate (no schema actions are enforced).
func (p *Processor) SetSchemaVersioner(versioner *SchemaVersioner) {
	if p == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.schemaVersioner = versioner
}

// SchemaHandshake answers a tablet's schema epoch. With no versioner attached
// the gate is open: the handshake reports SYNC at epoch 0, and No plan is returned.
func (p *Processor) SchemaHandshake(tabletEpoch SchemaEpoch) HandshakeResult {
	p.mu.RLock()
	versioner := p.schemaVersioner
	p.mu.RUnlock()
	if versioner == nil {
		return HandshakeResult{Status: StatusSync, ServerEpoch: 0, TabletEpoch: tabletEpoch}
	}
	return versioner.Handshake(tabletEpoch)
}
