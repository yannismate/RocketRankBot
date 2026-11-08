package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

func WithLogging(isInternal bool, base http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		traceId := r.Header.Get("trace-id")
		if traceId == "" || !isInternal {
			traceId = uuid.New().String()
		}
		spanId := uuid.New().String()

		ctx = context.WithValue(ctx, "trace-id", traceId)
		ctx = context.WithValue(ctx, "span-id", spanId)

		parentSpanId := ""
		if isInternal {
			parentSpanId = r.Header.Get("span-id")
			ctx = context.WithValue(ctx, "parent-span-id", parentSpanId)
		}

		ctxLogger := log.With().Str("trace-id", traceId).Str("parent-span-id", parentSpanId).Str("span-id", spanId).Logger()
		ctx = ctxLogger.WithContext(ctx)

		r = r.WithContext(ctx)
		base.ServeHTTP(w, r)

		log.Ctx(r.Context()).Trace().Msg(fmt.Sprint("[Server] ", r.Method, " ", r.URL.String()))
	})
}
