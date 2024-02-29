package metrics

import (
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
	"net/http"
)

func StartMetricsServer(bindAddress string, checkReadiness func() bool) {
	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.Handler())
		mux.HandleFunc("/health", func(res http.ResponseWriter, req *http.Request) {
			_, _ = res.Write([]byte("OK"))
		})
		mux.HandleFunc("/ready", func(res http.ResponseWriter, req *http.Request) {
			if checkReadiness() {
				_, _ = res.Write([]byte("OK"))
			} else {
				res.WriteHeader(http.StatusInternalServerError)
				_, _ = res.Write([]byte("Readiness check failed"))
			}
		})

		err := http.ListenAndServe(bindAddress, mux)
		if err != nil {
			log.Panic().Err(err).Msg("Metrics and k8s probe server failed")
			return
		}
	}()
}
