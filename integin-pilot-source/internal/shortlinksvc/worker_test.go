package shortlinksvc_test

import (
	"context"
	"testing"

	"github.com/riverqueue/river"

	"integin/internal/queue"
	"integin/internal/shortlinksvc"
)

func TestRiverWebhookDeliveryWorkerContract(t *testing.T) {
	// Verify that the worker satisfies River's worker interface contract
	worker := shortlinksvc.NewWebhookDeliveryWorker(nil)
	if worker == nil {
		t.Fatal("expected non-nil worker")
	}

	// Compile-time check that WebhookDeliveryJobArgs has Kind
	var args queue.WebhookDeliveryJobArgs
	if args.Kind() != "webhook_delivery_dispatch" {
		t.Fatalf("expected Kind 'webhook_delivery_dispatch', got %q", args.Kind())
	}

	// Verify River worker type assertion
	var _ river.Worker[queue.WebhookDeliveryJobArgs] = worker

	// Verify that Work returns without panic when context is canceled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	job := &river.Job[queue.WebhookDeliveryJobArgs]{
		Args: queue.WebhookDeliveryJobArgs{
			DeliveryID: 999,
			TenantID:   "test-tenant",
		},
	}
	_ = worker.Work(ctx, job)
}
