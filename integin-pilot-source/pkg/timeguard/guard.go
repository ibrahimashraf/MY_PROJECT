// Package timeguard protects time-dependent state against wall-clock
// tampering, CMOS rollbacks and offline clock skew (Hazard 33).
package timeguard

import (
	"errors"
	"sync"
	"time"
)

// TimeSource classifies the origin of a wall-clock sample.
type TimeSource string

const (
	GPS_ATTESTED    TimeSource = "GPS_ATTESTED"
	NTP_SYNCED      TimeSource = "NTP_SYNCED"
	MONOTONIC_LOCAL TimeSource = "MONOTONIC_LOCAL"
)

// MonotonicSample couples one wall-clock reading with an independent
// monotonic tick (Hazard 33: monotonic must never regress).
type MonotonicSample struct {
	WallTime      time.Time
	MonotonicNano int64
	Source        TimeSource
}

var (
	ErrClockBackwardsDrift     = errors.New("timeguard: wall clock drifted backwards beyond allowed skew")
	ErrClockForwardJump        = errors.New("timeguard: wall clock jumped forward beyond allowed skew")
	ErrMonotonicSequenceBroken = errors.New("timeguard: monotonic clock sequence decreased")
	ErrInvalidMonotonicSample  = errors.New("timeguard: sample wall time is zero")
)

// ClockGuard validates successive MonotonicSamples, anchoring on the first
// sample and rejecting backwards drift, rollback and forward jumps beyond
// the configured tolerance.
type ClockGuard struct {
	mu       sync.Mutex
	maxSkew  time.Duration
	lastWall time.Time
	lastMono int64
	init     bool
}

// NewClockGuard returns a guard tolerating up to maxAllowedBackwardsSkew of
// wall-clock deviation in either direction (a symmetric skew bound maps a
// single knob to both tamper rollback and NTP-style forward correction).
func NewClockGuard(maxAllowedBackwardsSkew time.Duration) *ClockGuard {
	if maxAllowedBackwardsSkew < 0 {
		maxAllowedBackwardsSkew = 0
	}
	return &ClockGuard{maxSkew: maxAllowedBackwardsSkew}
}

// ValidateSample anchors or checks a sample. MonotonicNano == 0 skips the
// monotonic check (callers without a tick source must not false-positive).
func (g *ClockGuard) ValidateSample(sample MonotonicSample) error {
	if sample.WallTime.IsZero() {
		return ErrInvalidMonotonicSample
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	if !g.init {
		g.lastWall = sample.WallTime
		g.lastMono = sample.MonotonicNano
		g.init = true
		return nil
	}

	if sample.MonotonicNano != 0 && sample.MonotonicNano < g.lastMono {
		return ErrMonotonicSequenceBroken
	}

	skew := sample.WallTime.Sub(g.lastWall)
	switch {
	case skew < 0:
		if -skew > g.maxSkew {
			return ErrClockBackwardsDrift
		}
	case skew > g.maxSkew:
		return ErrClockForwardJump
	}

	if skew > 0 {
		g.lastWall = sample.WallTime
	}
	g.lastMono = sample.MonotonicNano
	return nil
}
