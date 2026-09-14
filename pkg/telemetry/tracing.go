package telemetry

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strconv"
	"strings"
)

// TraceContext carries the W3C tracecontext fields encoded in a traceparent
// header (https://www.w3.org/TR/trace-context/). Only version 00 is parsed.
type TraceContext struct {
	TraceID string // 32 lowercase hex digits
	SpanID  string // 16 lowercase hex digits
	Sampled bool
}

var errInvalidTraceParent = errors.New("telemetry: invalid traceparent header")

// String renders the W3C traceparent header value.
func (tc TraceContext) String() string {
	flags := "00"
	if tc.Sampled {
		flags = "01"
	}
	return "00-" + tc.TraceID + "-" + tc.SpanID + "-" + flags
}

// ParseTraceParent parses a W3C traceparent header. Invalid or non-v00
// headers are rejected so downstream consumers never trust a malformed value.
func ParseTraceParent(value string) (TraceContext, error) {
	parts := strings.Split(value, "-")
	if len(parts) != 4 || parts[0] != "00" {
		return TraceContext{}, errInvalidTraceParent
	}
	flags := parts[3]
	if !validHex(parts[1], 16) || !validHex(parts[2], 8) || !validHex(flags, 1) {
		return TraceContext{}, errInvalidTraceParent
	}
	if allZero(parts[1]) || allZero(parts[2]) {
		return TraceContext{}, errInvalidTraceParent
	}
	flagByte, _ := strconv.ParseUint(flags, 16, 8)
	return TraceContext{TraceID: parts[1], SpanID: parts[2], Sampled: flagByte&0x01 == 0x01}, nil
}

func validHex(value string, length int) bool {
	if len(value) != length*2 {
		return false
	}
	for _, c := range value {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F') {
			return false
		}
	}
	return true
}

func allZero(value string) bool {
	for _, c := range value {
		if c != '0' {
			return false
		}
	}
	return true
}

// NewTraceContext generates a fresh random trace context.
func NewTraceContext() (TraceContext, error) {
	traceID, err := randomHex(16)
	if err != nil {
		return TraceContext{}, err
	}
	spanID, err := randomHex(8)
	if err != nil {
		return TraceContext{}, err
	}
	return TraceContext{TraceID: traceID, SpanID: spanID, Sampled: true}, nil
}

// WithNewSpan returns a copy of tc bound to a fresh child span ID.
func (tc TraceContext) WithNewSpan() (TraceContext, error) {
	spanID, err := randomHex(8)
	if err != nil {
		return TraceContext{}, err
	}
	tc.SpanID = spanID
	return tc, nil
}

func randomHex(bytes int) (string, error) {
	buf := make([]byte, bytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

type traceContextKey struct{}

// WithTraceContext attaches a trace context to ctx for propagation.
func WithTraceContext(ctx context.Context, tc TraceContext) context.Context {
	return context.WithValue(ctx, traceContextKey{}, tc)
}

// TraceContextFromContext retrieves the trace context attached to ctx.
func TraceContextFromContext(ctx context.Context) (TraceContext, bool) {
	tc, ok := ctx.Value(traceContextKey{}).(TraceContext)
	return tc, ok
}

var (
	// allowedTraceAttributes is the trace_contract.allowed_attributes list.
	allowedTraceAttributes = strSet(
		"service.name", "service.version", "deployment.environment",
		"http.route", "http.request.method", "http.response.status_code",
		"integin.operation", "integin.outcome", "error.type",
	)
	// prohibitedTraceAttributes is the trace_contract.prohibited_attributes
	// list, kept explicit for auditability even though the allowlist is the
	// enforced gate.
	prohibitedTraceAttributes = strSet(
		"http.request.body", "http.response.body", "db.statement",
		"db.connection_string", "enduser.id", "integin.tenant_id",
		"integin.organization_id", "integin.user_id", "integin.device_id",
		"integin.evidence_bytes", "integin.object_key", "integin.payload",
		"integin.token", "exception.message",
	)
)

// IsAllowedTraceAttribute reports whether an attribute key is permitted by the
// trace contract allowlist.
func IsAllowedTraceAttribute(key string) bool {
	_, ok := allowedTraceAttributes[key]
	return ok
}

// IsProhibitedTraceAttribute reports whether an attribute key is on the trace
// contract negative list.
func IsProhibitedTraceAttribute(key string) bool {
	_, ok := prohibitedTraceAttributes[key]
	return ok
}

// SafeTraceAttributes returns a copy of attrs containing only trace-contract
// allowed attributes. Prohibited and unknown attributes are dropped; the
// caller's map is never mutated.
func SafeTraceAttributes(attrs map[string]string) map[string]string {
	safe := make(map[string]string, len(attrs))
	for key, value := range attrs {
		if IsAllowedTraceAttribute(key) {
			safe[key] = value
		}
	}
	return safe
}
