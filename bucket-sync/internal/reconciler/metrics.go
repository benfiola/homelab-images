package reconciler

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

const (
	ResultSucceeded = "succeeded"
	ResultFailed    = "failed"

	TriggerManual   = "manual"
	TriggerSchedule = "schedule"
)

var (
	metricSyncsCompleted = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "bucket_sync_syncs_completed_total",
		Help: "Total number of bucket syncs that reached a terminal phase, by result and policy (empty if created without one).",
	}, []string{"result", "policy"})

	metricPolicyConsecutiveFailures = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "bucket_sync_policy_consecutive_failures",
		Help: "Number of consecutive failed syncs for a policy since its last success. Resets to 0 on success and when the controller restarts.",
	}, []string{"policy"})

	metricSyncDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "bucket_sync_sync_duration_seconds",
		Help:    "Time between a bucket sync starting and finishing, by result.",
		Buckets: []float64{15, 30, 60, 120, 300, 600, 1800, 3600},
	}, []string{"result"})

	metricLockWaits = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "bucket_sync_lock_waits_total",
		Help: "Total number of times a bucket sync had to wait on a lock held by another sync.",
	})

	metricStaleLocksDeleted = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "bucket_sync_stale_locks_deleted_total",
		Help: "Total number of stale lock config maps deleted, by reason.",
	}, []string{"reason"})

	metricPolicySyncsTriggered = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "bucket_sync_policy_syncs_triggered_total",
		Help: "Total number of bucket syncs created by policies, by trigger.",
	}, []string{"trigger"})
)

func init() {
	metrics.Registry.MustRegister(
		metricSyncsCompleted,
		metricSyncDuration,
		metricPolicyConsecutiveFailures,
		metricLockWaits,
		metricStaleLocksDeleted,
		metricPolicySyncsTriggered,
	)
}
