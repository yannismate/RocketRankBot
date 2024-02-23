package server

import (
	"context"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/twitchtv/twirp"
	"net/http"
)

func WithLogging(base http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		traceId := r.Header.Get("trace-id")
		if traceId == "" {
			traceId = uuid.New().String()
		}
		parentSpanId := r.Header.Get("span-id")
		spanId := uuid.New().String()

		ctx = context.WithValue(ctx, "trace-id", traceId)
		ctx = context.WithValue(ctx, "parent-span-id", parentSpanId)
		ctx = context.WithValue(ctx, "span-id", spanId)

		ctxLogger := log.With().Str("trace-id", traceId).Str("parent-span-id", parentSpanId).Str("span-id", spanId).Logger()
		ctx = ctxLogger.WithContext(ctx)

		outgoingHeaders := make(http.Header)
		outgoingHeaders.Set("trace-id", traceId)
		outgoingHeaders.Set("span-id", spanId)
		ctx, err := twirp.WithHTTPRequestHeaders(ctx, outgoingHeaders)
		if err != nil {
			ctxLogger.Panic().Err(err)
		}

		r = r.WithContext(ctx)
		base.ServeHTTP(w, r)
	})
}
