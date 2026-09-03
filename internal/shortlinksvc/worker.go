package shortlinksvc

import (
	"context"
	"log/slog"
	"time"
)

// RetryWorker processes pending webhook deliveries on a configurable interval.
type RetryWorker struct {
	svc             *Service
	interval        time.Duration
	cleanupInterval time.Duration
	stopCh          chan struct{}
}

// NewRetryWorker creates a new RetryWorker with the given service and interval.
func NewRetryWorker(svc *Service, interval time.Duration) *RetryWorker {
	return &RetryWorker{
		svc:             svc,
		interval:        interval,
		cleanupInterval: 1 * time.Hour,
		stopCh:          make(chan struct{}),
	}
}

// Start begins the retry worker loop.
func (w *RetryWorker) Start(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(w.interval)
		defer ticker.Stop()
		cleanupTicker := time.NewTicker(w.cleanupInterval)
		defer cleanupTicker.Stop()

		for {
			select {
			case <-w.stopCh:
				return
			case <-ctx.Done():
				return
			case <-ticker.C:
				procCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
				if err := w.svc.ProcessWebhookRetries(procCtx); err != nil {
					slog.Error("webhook retry processing failed", "error", err)
				}
				cancel()
			case <-cleanupTicker.C:
				cleanCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
				n, err := w.svc.CleanupExpiredLinks(cleanCtx, 24*time.Hour, 1000)
				if err != nil {
					slog.Error("expired link cleanup failed", "error", err)
				} else if n > 0 {
					slog.Info("expired links cleaned", "count", n)
				}
				cancel()
			}
		}
	}()
}

// Stop signals the worker to stop.
func (w *RetryWorker) Stop() {
	close(w.stopCh)
}
