package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	CounterReceivedMessages = promauto.NewCounter(prometheus.CounterOpts{
		Name: "twitchconnector_received_messages",
		Help: "Number of received IRC messages",
	})
	CounterSentMessages = promauto.NewCounter(prometheus.CounterOpts{
		Name: "twitchconnector_sent_messages",
		Help: "Number of sent IRC messages",
	})
	CounterReceivedPossibleCommands = promauto.NewCounter(prometheus.CounterOpts{
		Name: "twitchconnector_received_possible_commands",
		Help: "Number of received IRC messages starting with command prefix",
	})
	GaugeJoinedChannels = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "twitchconnector_joined_channels",
		Help: "Number of joined IRC channels",
	})
)
