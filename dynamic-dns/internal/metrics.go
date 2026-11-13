package internal

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"

	"github.com/benfiola/homelab-images/shared/pkg/logging"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	StagePublicIP     = "public_ip"
	StageFindZone     = "find_zone"
	StageFindRecord   = "find_record"
	StageUpdateRecord = "update_record"
)

var (
	metricSyncs = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "dynamic_dns_syncs_total",
		Help: "Total number of sync attempts, by result.",
	}, []string{"result"})

	metricSyncDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "dynamic_dns_sync_duration_seconds",
		Help:    "Time taken to complete a sync.",
		Buckets: []float64{0.5, 1, 2.5, 5, 10, 30, 60},
	})

	metricSyncErrors = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "dynamic_dns_sync_errors_total",
		Help: "Total number of errors encountered during syncs, by stage.",
	}, []string{"stage"})

	metricRecordUpdates = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "dynamic_dns_record_updates_total",
		Help: "Total number of dns records updated to a new ip address.",
	})

	metricLastSuccess = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "dynamic_dns_last_successful_sync_timestamp_seconds",
		Help: "Unix timestamp of the last fully successful sync.",
	})
)

func init() {
	metricSyncs.WithLabelValues("success")
	metricSyncs.WithLabelValues("failure")
	for _, stage := range []string{StagePublicIP, StageFindZone, StageFindRecord, StageUpdateRecord} {
		metricSyncErrors.WithLabelValues(stage)
	}

	prometheus.MustRegister(
		metricSyncs,
		metricSyncDuration,
		metricSyncErrors,
		metricRecordUpdates,
		metricLastSuccess,
	)
}

func recordSync(start time.Time, err error) {
	metricSyncDuration.Observe(time.Since(start).Seconds())
	if err != nil {
		metricSyncs.WithLabelValues("failure").Inc()
		return
	}
	metricSyncs.WithLabelValues("success").Inc()
	metricLastSuccess.SetToCurrentTime()
}

func serveMetrics(ctx context.Context, address string) error {
	logger := logging.FromContext(ctx)

	listener, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	srv := &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv.Shutdown(shutdownCtx)
	}()

	go func() {
		logger.Info("serving metrics", "address", address)
		err := srv.Serve(listener)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("metrics server failed", "error", err)
		}
	}()

	return nil
}
