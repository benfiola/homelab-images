package reconciler

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

const (
	ResultSucceeded = "succeeded"
	ResultFailed    = "failed"
)

var (
	metricRestoresCompleted = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "pvc_restore_restores_completed_total",
		Help: "Total number of pvc restores that reached a terminal phase, by result.",
	}, []string{"result"})

	metricRestoreDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "pvc_restore_restore_duration_seconds",
		Help:    "Time between a pvc restore being created and finishing, by result.",
		Buckets: []float64{15, 30, 60, 120, 300, 600, 1800, 3600},
	}, []string{"result"})

	metricLockWaits = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "pvc_restore_lock_waits_total",
		Help: "Total number of times a pvc restore had to wait on a lock held by another restore.",
	})

	metricStaleLocksDeleted = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "pvc_restore_stale_locks_deleted_total",
		Help: "Total number of stale lock config maps deleted, by reason.",
	}, []string{"reason"})

	metricRestoresRequested = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "pvc_restore_restores_requested_total",
		Help: "Total number of pvc restores created from pvc annotations.",
	})
)

func init() {
	metrics.Registry.MustRegister(
		metricRestoresCompleted,
		metricRestoreDuration,
		metricLockWaits,
		metricStaleLocksDeleted,
		metricRestoresRequested,
	)
}
