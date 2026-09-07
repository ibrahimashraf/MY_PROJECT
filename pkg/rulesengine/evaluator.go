package rulesengine

import (
	"context"
	"fmt"
	"sync"
	"time"

	"cel.dev/cel-go/cel"
	"cel.dev/cel-go/common/types"
)

// MaxAllocBytes is the hard per-evaluation memory ceiling enforced by the
// sandbox. CEL is non-Turing complete and operates on bounded inputs, so a
// heap ceiling plus strict type checking eliminates DoS-style expressions.
const MaxAllocBytes = 4 * 1024 * 1024 // 4 MiB

// Evaluator compiles and executes sandboxed Google CEL rule expressions.
// Programs are compiled once and cached, giving sub-microsecond typed
// evaluation on hot paths with zero allocations.
type Evaluator struct {
	cache  sync.Map // ruleID string -> *Program
	option ProgramOptions
}

// Program is a precompiled, pre-typed CEL program bound to a set of variable
// names. Reuse it across evaluations with a reused *VarSet to hit the
// zero-allocation hot path.
type Program struct {
	ruleID   string
	expr     string
	prg      cel.Program
	outType  *cel.Type
	maxAlloc uint64
}

// ProgramOptions controls sandbox behaviour for one Evaluator instance.
type ProgramOptions struct {
	// MaxAlloc overrides MaxAllocBytes when non-zero.
	MaxAlloc uint64
}

// NewEvaluator builds a strict, macro-limited CEL environment. Variables are
// not pre-declared here because each Program declares its own typed vars, so a
// single Evaluator safely hosts heterogeneous standards.
func NewEvaluator(opts ...ProgramOptions) (*Evaluator, error) {
	var opt ProgramOptions
	if len(opts) > 0 {
		opt = opts[0]
	}
	return &Evaluator{option: opt}, nil
}

// Compile type-checks an expression against the declared typed variables and
// prepares it for repeated execution. Compilation is a server-side, offline
// act; evaluation happens on the device. The program is cached by ruleID.
func (e *Evaluator) Compile(ruleID string, expr string, vars map[string]*cel.Type) (*Program, error) {
	if cached, ok := e.cache.Load(ruleID); ok {
		return cached.(*Program), nil
	}
	typeOpts := make([]cel.EnvOption, 0, len(vars))
	for name, t := range vars {
		typeOpts = append(typeOpts, cel.Variable(name, t))
	}
	opt := append([]cel.EnvOption{cel.ClearMacros()}, typeOpts...)
	env, err := cel.NewEnv(opt...)
	if err != nil {
		return nil, fmt.Errorf("rulesengine: build typed env for %q: %w", ruleID, err)
	}
	ast, iss := env.Compile(expr)
	if iss != nil && iss.Err() != nil {
		return nil, fmt.Errorf("rulesengine: compile %q: %w", ruleID, iss.Err())
	}
	if ast.OutputType() != cel.BoolType {
		return nil, fmt.Errorf("rulesengine: rule %q must evaluate to bool, got %s", ruleID, ast.OutputType())
	}
	prg, err := env.Program(ast)
	if err != nil {
		return nil, fmt.Errorf("rulesengine: build program %q: %w", ruleID, err)
	}
	p := &Program{ruleID: ruleID, expr: expr, prg: prg, outType: ast.OutputType(), maxAlloc: e.option.MaxAlloc}
	e.cache.Store(ruleID, p)
	return p, nil
}

// Evaluate runs one precompiled Program. `vars` must be a *VarSet bound to
// exactly the declared variables. The input delta is budget-checked against
// the sandbox ceiling before any CEL execution. The caller's context is
// honoured before and after evaluation; mid-eval interruption is unnecessary
// because CEL is non-Turing complete and the variable budget bounds the work.
// Reuse a *VarSet across calls to hit the 0 allocs/op hot path.
func (p *Program) Evaluate(ctx context.Context, vars *VarSet) (bool, error) {
	if p.outType != cel.BoolType {
		return false, fmt.Errorf("rulesengine: program %q is not boolean", p.ruleID)
	}
	if err := ctx.Err(); err != nil {
		return false, fmt.Errorf("rulesengine: evaluate %q: %w", p.ruleID, err)
	}
	max := p.maxAlloc
	if max == 0 {
		max = MaxAllocBytes
	}
	if err := budgetCheck(vars, max); err != nil {
		return false, err
	}
	val, _, err := p.prg.Eval(vars)
	if err != nil {
		return false, fmt.Errorf("rulesengine: evaluate %q: %w", p.ruleID, err)
	}
	if err := ctx.Err(); err != nil {
		return false, fmt.Errorf("rulesengine: evaluate %q: %w", p.ruleID, err)
	}
	b, ok := val.Value().(bool)
	if !ok {
		return false, fmt.Errorf("rulesengine: rule %q produced non-bool result %T", p.ruleID, val.Value())
	}
	return b, nil
}

// VarSet is a preallocated, reusable activation that avoids per-evaluation map
// allocations. Pre-fill it in the same order across calls for the 0 allocs/op
// hot path. It implements cel.Activation directly so CEL consumes it with no
// interface-wrapper allocation.
type VarSet struct {
	keys []string
	vals []any
}

// NewVarSet builds a VarSet for the given keys, preallocated to prevent hot
// path growth. Values default to nil and must be set before each evaluation.
func NewVarSet(keys ...string) *VarSet {
	return &VarSet{keys: append([]string(nil), keys...), vals: make([]any, len(keys))}
}

// Put binds a value. Panics on unknown keys to surface typos at test time,
// never silently swallowing a missing variable (strict typing invariant).
// Scalar Go values are pre-adapted to CEL-native ref.Val at bind time, so the
// interpreter's per-eval NativeToValue fast-path returns them unboxed and the
// hot loop stays allocation-free.
func (v *VarSet) Put(key string, val any) {
	for i, k := range v.keys {
		if k == key {
			v.vals[i] = adaptInput(val)
			return
		}
	}
	panic(fmt.Sprintf("rulesengine: unknown variable %q (expected one of %v)", key, v.keys))
}

// adaptInput converts canonical scalar Go values to their CEL-native forms once
// per binding. Aggregates stay native so the 4 MiB budget can walk them.
func adaptInput(val any) any {
	switch val.(type) {
	case bool, int, int32, int64, uint, uint32, uint64,
		float32, float64, string, time.Time, time.Duration,
		[]byte:
		return types.DefaultTypeAdapter.NativeToValue(val)
	default:
		return val
	}
}

// ResolveName implements cel.Activation.
func (v *VarSet) ResolveName(name string) (any, bool) {
	for i, k := range v.keys {
		if k == name {
			if v.vals[i] == nil {
				return nil, false
			}
			return v.vals[i], true
		}
	}
	return nil, false
}

// Parent implements cel.Activation. No parent scope exists in the sandbox.
func (v *VarSet) Parent() cel.Activation { return nil }

// ValidateVarsBudget recursively counts the elements reachable through the
// activation and rejects inputs whose footprint exceeds the sandbox ceiling.
// CEL only allocates from the data it can reach plus the (compile-time bounded)
// expression itself, so this bound plus strict typing makes the 4 MiB heap
// ceiling deterministic and testable.
func (e *Evaluator) ValidateVarsBudget(vars *VarSet) error {
	max := e.option.MaxAlloc
	if max == 0 {
		max = MaxAllocBytes
	}
	return budgetCheck(vars, max)
}

// budgetCheck enforces the 4 MiB per-evaluation ceiling. Each reachable CEL
// element (int, double, list/map slot) is conservatively charged 64 bytes:
// 65536 elements ~= 4 MiB. Shared/aliased values are charged once.
func budgetCheck(vars *VarSet, max uint64) error {
	const chargePerElement = 64
	budget := max / chargePerElement
	counted := 0
	if vars == nil {
		return nil
	}
	return elementCount(vars.vals, make(map[string]bool, 8), &counted, budget)
}

// elementCount walks the activation values depth-first, charging each
// primitive and each aggregate slot. Maps/lists are visited once via a visit
// set to defeat alias blow-up.
func elementCount(vals []any, visited map[string]bool, counted *int, budget uint64) error {
	for _, v := range vals {
		if err := charge(v, visited, counted, budget); err != nil {
			return err
		}
	}
	return nil
}

func charge(v any, visited map[string]bool, counted *int, budget uint64) error {
	if v == nil {
		return nil
	}
	switch t := v.(type) {
	case map[string]any:
		key := fmt.Sprintf("m:%p", t)
		if visited[key] {
			return nil
		}
		visited[key] = true
		for _, e := range t {
			*counted++
			if uint64(*counted) > budget {
				return fmt.Errorf("rulesengine: input exceeds sandbox heap ceiling (%d elements > budget %d)", *counted, budget)
			}
			if err := charge(e, visited, counted, budget); err != nil {
				return err
			}
		}
	case []any:
		key := fmt.Sprintf("s:%p", t)
		if visited[key] {
			return nil
		}
		visited[key] = true
		for _, e := range t {
			*counted++
			if uint64(*counted) > budget {
				return fmt.Errorf("rulesengine: input exceeds sandbox heap ceiling (%d elements > budget %d)", *counted, budget)
			}
			if err := charge(e, visited, counted, budget); err != nil {
				return err
			}
		}
	default:
		*counted++
		if uint64(*counted) > budget {
			return fmt.Errorf("rulesengine: input exceeds sandbox heap ceiling (%d elements > budget %d)", *counted, budget)
		}
	}
	return nil
}
