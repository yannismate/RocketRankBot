package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
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
)
