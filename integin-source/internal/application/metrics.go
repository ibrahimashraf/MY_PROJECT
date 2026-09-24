package application

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// MetricStateTransitions counts every successful state evaluation (Edge and Web).
	MetricStateTransitions = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "assurance_state_transitions_total",
		Help: "The total number of successful assurance state transitions",
	}, []string{"tenant_id", "state", "is_offline_origin"})

	// MetricSignatureTriage counts triage outcomes (accepted, tampered, quarantined).
	MetricSignatureTriage = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "assurance_signature_triage_total",
		Help: "The total number of cryptographic triage outcomes",
	}, []string{"tenant_id", "result"})

	// MetricSyncConflicts counts lamport causality overwrites or drops.
	MetricSyncConflicts = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "assurance_sync_conflicts_total",
		Help: "The total number of edge sync causal ordering conflicts resolved",
	}, []string{"tenant_id", "resolution"})
)
