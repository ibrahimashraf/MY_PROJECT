package shortlinksvc

import (
	"context"
	"log/slog"
	"time"
)

type RetryWorker struct {
	svc      *Service
	interval time.Duration
	stopCh   chan struct{}
}

func NewRetryWorker(svc *Service, interval time.Duration) *RetryWorker {
	return &RetryWorker{
		svc:      svc,
		interval: interval,
		stopCh:   make(chan struct{}),
	}
}

func (w *RetryWorker) Start(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(w.interval)
		defer ticker.Stop()
		for {
			select {
			case <-w.stopCh:
				return
case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			if err := w.svc.ProcessWebhookRetries(ctx); err != nil {
				slog.Error("webhook retry processing failed", "error", err)
			}
			cancel()
			}
		}
	}()
}

func (w *RetryWorker) Stop() {
	close(w.stopCh)
}