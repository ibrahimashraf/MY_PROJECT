package rulesengine

import (
	"fmt"
	"time"
)

// RuleSetState is the lifecycle state of a ruleset revision.
type RuleSetState string

const (
	StateDRAFT      RuleSetState = "DRAFT"
	StateACTIVE     RuleSetState = "ACTIVE"
	StateDEPRECATED RuleSetState = "DEPRECATED"
)

// validTransitions guards the state machine. A stateset cannot skip DRAFT or
// re-activate a DEPRECATED revision, and activation timestamps are immutable.
var validTransitions = map[RuleSetState]map[RuleSetState]bool{
	StateDRAFT:  {StateACTIVE: true},
	StateACTIVE: {StateDEPRECATED: true},
}

// Activate commits a DRAFT ruleset to ACTIVE with an immutable activation
// timestamp. Re-activating, deprecating a DRAFT, or transitioning from an
// invalid state is rejected.
func (rs *RuleSet) Activate(now time.Time) error {
	if err := rs.maybeSetActivatedAt(now); err != nil {
		return err
	}
	if !validTransitions[rs.State][StateACTIVE] {
		return fmt.Errorf("ruleset %s revision %d: illegal transition %s -> ACTIVE", rs.ID, rs.Revision, rs.State)
	}
	rs.State = StateACTIVE
	return nil
}

// Deprecate retires an ACTIVE ruleset. Activation timestamp is preserved.
func (rs *RuleSet) Deprecate() error {
	if !validTransitions[rs.State][StateDEPRECATED] {
		return fmt.Errorf("ruleset %s revision %d: illegal transition %s -> DEPRECATED (only ACTIVE may deprecate)", rs.ID, rs.Revision, rs.State)
	}
	rs.State = StateDEPRECATED
	return nil
}

func (rs *RuleSet) maybeSetActivatedAt(now time.Time) error {
	if !rs.ActivatedAt.IsZero() {
		return fmt.Errorf("ruleset %s revision %d: activation timestamp is immutable (%s)", rs.ID, rs.Revision, rs.ActivatedAt.Format(time.RFC3339))
	}
	rs.ActivatedAt = now
	return nil
}
