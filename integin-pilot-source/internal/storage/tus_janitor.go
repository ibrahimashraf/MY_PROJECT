package storage

import (
	"context"
	"errors"
	leimenv "integin/internal/shared/env"
	"log/slog"
	"time"
)

// DefaultJanitorInterval is how often RunJanitor sweeps upload state when the
// caller does not supply an explicit interval.
const DefaultJanitorInterval = 10 * time.Minute

// JanitorReport is the outcome of one janitor sweep. It is the metrics surface
// for upload hygiene: how many crashed sessions were resumed, how many
// orphaned chunk files were deleted, how many stale live sessions were pruned,
// and how many uploads remain active.
type JanitorReport struct {
	Resumed int
	Deleted int
	Purged  int
	Active  int
}

// Sweep runs crash recovery and stale pruning in one pass and emits the
// aggregate counts as a structured cleanup log event. It is safe to call
// repeatedly and is the unit of work behind RunJanitor.
func (m *TUSManager) Sweep(ctx context.Context, logger *slog.Logger) (JanitorReport, error) {
	if logger == nil {
		logger = slog.Default()
	}
	if err := ctx.Err(); err != nil {
		return JanitorReport{}, err
	}
	resumed, deleted, err := m.RecoverOrphans()
	if err != nil {
		return JanitorReport{}, err
	}
	purged, err := m.PurgeStale(ctx)
	if err != nil {
		return JanitorReport{}, err
	}
	m.mu.RLock()
	active := len(m.sessions)
	m.mu.RUnlock()
	logger.Info("tus janitor sweep",
		"resumed", resumed, "orphan_deleted", deleted, "stale_purged", purged, "active_uploads", active)
	return JanitorReport{Resumed: resumed, Deleted: deleted, Purged: purged, Active: active}, nil
}

// RunJanitor is a blocking periodic sweeper until ctx is cancelled. It sweeps
// once immediately (so startup recovery is not delayed by the ticker), then on
// every interval. Periodic failures are logged and the loop keeps running so a
// transient filesystem problem cannot halt upload hygiene.
func (m *TUSManager) RunJanitor(ctx context.Context, interval time.Duration, logger *slog.Logger) error {
	if logger == nil {
		logger = slog.Default()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if interval <= 0 {
		interval = leimenv.Seconds("INTEGIN_TUS_JANITOR_INTERVAL_S", DefaultJanitorInterval, 60, 3600)
	}
	if _, err := m.Sweep(ctx, logger); err != nil {
		return err
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if _, err := m.Sweep(ctx, logger); err != nil && !errors.Is(err, context.Canceled) {
				logger.Error("tus janitor sweep failed", "error", err)
			}
		}
	}
}
