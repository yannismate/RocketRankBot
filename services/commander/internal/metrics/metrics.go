package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	CounterWebHookNotifications = promauto.NewCounter(prometheus.CounterOpts{
		Name: "commander_webhook_notifications",
		Help: "Number of received valid webhook notifications",
	})
	CounterExecutedCommandsBuiltin = promauto.NewCounter(prometheus.CounterOpts{
		Name: "commander_commands_builtin_total",
		Help: "Number of executed builtin commands",
	})
	CounterExecutedCommandsRank = promauto.NewCounter(prometheus.CounterOpts{
		Name: "commander_commands_rank_total",
		Help: "Number of executed rank commands",
	})
	CounterCachedCommandsRank = promauto.NewCounter(prometheus.CounterOpts{
		Name: "commander_commands_rank_cached",
		Help: "Number of cached rank commands",
	})
	CounterCachedRequestsRank = promauto.NewCounter(prometheus.CounterOpts{
		Name: "commander_requests_rank_cached",
		Help: "Number of cached rank requests",
	})
	HistogramCommandResponseTime = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "commander_commands_response_time",
		Help: "Number of cached rank requests",
	}, []string{"type"})
	HistogramTrackerRequestTime = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "commander_tracker_request_duration_seconds",
		Help:    "Duration of rank lookups in seconds, including unsuccessful lookups",
		Buckets: prometheus.ExponentialBuckets(0.1, 2, 12),
	})
	CounterTrackerRequests = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "commander_tracker_requests_total",
		Help: "Number of rank lookups by outcome and error code",
	}, []string{"outcome", "error_code"})
	GaugeTrackerPrewarmedSessions = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "commander_tracker_prewarmed_sessions",
		Help: "Number of successfully prewarmed, unexpired sessions, including busy sessions",
	})
	HistogramTrackerSessionRequests = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "commander_tracker_session_requests_on_replacement",
		Help:    "Requests served by each replaced session, including prewarming and failed requests",
		Buckets: prometheus.LinearBuckets(0, 10, 12),
	})
)
