package util

import (
	"fmt"
	"github.com/rs/zerolog/log"
	"net/http"
)

type LoggingRoundTripper struct{}

func (lrt *LoggingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	res, err := http.DefaultTransport.RoundTrip(req)
	if err == nil {
		if req.Context().Value("trace-id") != nil {
			log.Ctx(req.Context()).Info().Msg(fmt.Sprint("[HTTP Client] ", req.Method, " ", req.URL.String(), " - ", res.StatusCode))
		} else {
			log.Info().Msg(fmt.Sprint("[HTTP Client] ", req.Method, " ", req.URL.String(), " - ", res.StatusCode))
		}
	}
	return res, err
}
