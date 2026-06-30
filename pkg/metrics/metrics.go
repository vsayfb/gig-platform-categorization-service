package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	HttpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "core_http_requests_total",
			Help: "Total HTTP requests handled, by route, method and status.",
		},
		[]string{"route", "method", "status"},
	)

	HttpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "core_http_request_duration_seconds",
			Help:    "HTTP request latency in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"route", "method", "status"},
	)

	WorkerMessagesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "worker_messages_processed_total",
			Help: "Total number of processed worker messages.",
		},
		[]string{"outcome"},
	)

	WorkerProcessingDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "worker_message_processing_duration_seconds",
			Help:    "Time spent processing worker messages.",
			Buckets: prometheus.DefBuckets,
		},
	)
)

func Register() {
	prometheus.MustRegister(
		HttpRequestsTotal,
		HttpRequestDuration,
		WorkerMessagesTotal,
		WorkerProcessingDuration,
	)
}

func ObserveWorkerProcessing(start time.Time, err error) {
	WorkerProcessingDuration.Observe(time.Since(start).Seconds())

	outcome := "success"
	if err != nil {
		outcome = "failure"
	}

	WorkerMessagesTotal.WithLabelValues(outcome).Inc()
}
