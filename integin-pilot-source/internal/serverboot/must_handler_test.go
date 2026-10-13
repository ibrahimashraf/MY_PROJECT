package serverboot

import (
	"net/http"
	"testing"
)

// mustHandler must pass the constructed value through untouched on success.
// Its error path calls log.Fatal and exits the process, so only the success
// path is exercisable in-process; the failure behaviour is asserted by the
// boot sequence itself (a route that cannot be wired must stop the process
// rather than mount as a nil handler).
func TestMustHandlerPassesValueThroughOnSuccess(t *testing.T) {
	if got := mustHandler(http.NotFoundHandler(), nil); got == nil {
		t.Fatal("mustHandler returned nil for a successful construction")
	}

	if got := mustHandler(42, nil); got != 42 {
		t.Fatalf("mustHandler value = %d, want 42", got)
	}
}
