package timeguard

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

var anchor = MonotonicSample{
	WallTime:      time.UnixMilli(1700000000000).UTC(),
	MonotonicNano: 1_000_000,
	Source:        GPS_ATTESTED,
}

func mustSample(wallShift, monoShift time.Duration) MonotonicSample {
	return MonotonicSample{
		WallTime:      anchor.WallTime.Add(wallShift),
		MonotonicNano: anchor.MonotonicNano + int64(monoShift),
		Source:        NTP_SYNCED,
	}
}

func TestClockGuardAcceptsSteadyProgress(t *testing.T) {
	g := NewClockGuard(time.Second)
	if err := g.ValidateSample(anchor); err != nil {
		t.Fatalf("anchor reject: %v", err)
	}
	if err := g.ValidateSample(mustSample(50*time.Millisecond, 50*time.Millisecond)); err != nil {
		t.Fatalf("steady progress reject: %v", err)
	}
}

func TestClockGuardSkewWithinTolerance(t *testing.T) {
	g := NewClockGuard(5 * time.Millisecond)
	if err := g.ValidateSample(anchor); err != nil {
		t.Fatalf("anchor reject: %v", err)
	}
	back := mustSample(-100*time.Microsecond, 1*time.Millisecond)
	if err := g.ValidateSample(back); err != nil {
		t.Fatalf("small backward skew reject: %v", err)
	}
	// Anchor must not regress: a small forward step after the back-step is fine.
	if err := g.ValidateSample(mustSample(2*time.Millisecond, 2*time.Millisecond)); err != nil {
		t.Fatalf("forward follow-up reject: %v", err)
	}
}

func TestClockGuardRollbackBeyondTolerance(t *testing.T) {
	g := NewClockGuard(100 * time.Millisecond)
	if err := g.ValidateSample(anchor); err != nil {
		t.Fatalf("anchor reject: %v", err)
	}
	err := g.ValidateSample(mustSample(-time.Second, 10*time.Millisecond))
	if !errors.Is(err, ErrClockBackwardsDrift) {
		t.Fatalf("err = %v, want ErrClockBackwardsDrift", err)
	}
}

func TestClockGuardJumpForward(t *testing.T) {
	g := NewClockGuard(time.Second)
	if err := g.ValidateSample(anchor); err != nil {
		t.Fatalf("anchor reject: %v", err)
	}
	err := g.ValidateSample(mustSample(10*time.Second, 10*time.Second))
	if !errors.Is(err, ErrClockForwardJump) {
		t.Fatalf("err = %v, want ErrClockForwardJump", err)
	}
}

func TestClockGuardMonotonicSequenceBroken(t *testing.T) {
	g := NewClockGuard(time.Second)
	if err := g.ValidateSample(anchor); err != nil {
		t.Fatalf("anchor reject: %v", err)
	}
	err := g.ValidateSample(mustSample(10*time.Millisecond, -500)) // wall advances, mono regresses
	if !errors.Is(err, ErrMonotonicSequenceBroken) {
		t.Fatalf("err = %v, want ErrMonotonicSequenceBroken", err)
	}
}

func TestClockGuardZeroMonotonicSkipped(t *testing.T) {
	g := NewClockGuard(time.Second)
	if err := g.ValidateSample(anchor); err != nil {
		t.Fatalf("anchor reject: %v", err)
	}
	if err := g.ValidateSample(mustSample(10*time.Millisecond, 0)); err != nil {
		t.Fatalf("zero monotonic must be skipped, got: %v", err)
	}
}

func TestClockGuardRejectsZeroWallTime(t *testing.T) {
	g := NewClockGuard(time.Second)
	err := g.ValidateSample(MonotonicSample{})
	if !errors.Is(err, ErrInvalidMonotonicSample) {
		t.Fatalf("err = %v, want ErrInvalidMonotonicSample", err)
	}
}

func TestClockGuardStateUncorruptedAfterRejection(t *testing.T) {
	g := NewClockGuard(100 * time.Millisecond)
	if err := g.ValidateSample(anchor); err != nil {
		t.Fatalf("anchor reject: %v", err)
	}
	if err := g.ValidateSample(mustSample(-time.Second, 10*time.Millisecond)); !errors.Is(err, ErrClockBackwardsDrift) {
		t.Fatalf("err = %v, want ErrClockBackwardsDrift", err)
	}
	if err := g.ValidateSample(mustSample(10*time.Millisecond, 10*time.Millisecond)); err != nil {
		t.Fatalf("valid follow-up after rejection failed: %v", err)
	}
}

func TestClockGuardConcurrent(t *testing.T) {
	g := NewClockGuard(time.Second)
	var wg sync.WaitGroup
	var tick atomic.Int64
	const samplesPerGoroutine, goroutines = 2000, 8
	for w := 0; w < goroutines; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < samplesPerGoroutine; i++ {
				value := tick.Add(1) // globally unique, increasing per-goroutine stream
				sample := MonotonicSample{
					WallTime:      time.UnixMilli(1700000000000 + value),
					MonotonicNano: value * int64(time.Millisecond),
					Source:        MONOTONIC_LOCAL,
				}
				if err := g.ValidateSample(sample); err != nil && !errors.Is(err, ErrMonotonicSequenceBroken) {
					// Cross-stream interleaving may legitimately present a
					// lower value after a higher one, which must surface only
					// as ErrMonotonicSequenceBroken (checked before wall).
					t.Errorf("concurrent sample rejected with unexpected error: %v", err)
					return
				}
			}
		}()
	}
	wg.Wait()
}
