package main

import (
	"RocketRankBot/services/commander/internal/bot"
	"RocketRankBot/services/commander/internal/config"
	"RocketRankBot/services/commander/internal/db"
	"RocketRankBot/services/commander/internal/metrics"
	"RocketRankBot/services/commander/internal/server"
	"RocketRankBot/services/commander/internal/tracker"
	"RocketRankBot/services/commander/internal/twitch"
	"context"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

func main() {
	cfg, err := config.ReadConfig("config.json")
	if err != nil {
		log.Fatal().Err(err).Msg("Config could not be read")
		return
	}

	mainDB, err := db.NewMainDB(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("Could not connect to main database")
		return
	}

	cacheDB, err := db.NewCache(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("Could not connect to cache database")
		return
	}

	metrics.StartMetricsServer(":"+strconv.Itoa(cfg.AdminPort), func() bool {
		return mainDB.IsConnected() && cacheDB.IsConnected()
	})

	ctx, cancel := signal.NotifyContext(newRootContext(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	ranks := tracker.New(ctx, cfg.FlareSolverr.URL, time.Duration(cfg.FlareSolverr.SessionMaxAgeSeconds)*time.Second)
	defer ranks.Close()
	twitchAPI := twitch.NewAPI(cfg, cacheDB)

	botInstance := bot.NewBot(mainDB, cacheDB, cfg, twitchAPI, ranks)

	serverInstance := server.NewServer(cfg, twitchAPI, mainDB, cacheDB, botInstance)
	err = serverInstance.Start(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Could not start server instance!")
		return
	}

	<-ctx.Done()
}

func newRootContext() context.Context {
	ctx := context.Background()

	traceId := uuid.New().String()
	spanId := uuid.New().String()
	ctx = context.WithValue(ctx, "trace-id", traceId)
	ctx = context.WithValue(ctx, "span-id", spanId)

	ctxLogger := log.With().Str("trace-id", traceId).Str("span-id", spanId).Logger()
	ctx = ctxLogger.WithContext(ctx)

	return ctx
}
