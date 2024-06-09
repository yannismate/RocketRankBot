package main

import (
	"RocketRankBot/services/commander/internal/bot"
	"RocketRankBot/services/commander/internal/config"
	"RocketRankBot/services/commander/internal/db"
	"RocketRankBot/services/commander/internal/metrics"
	"RocketRankBot/services/commander/internal/server"
	"RocketRankBot/services/commander/internal/twitch"
	"RocketRankBot/services/commander/internal/util"
	"RocketRankBot/services/commander/rpc/trackerggscraper"
	"context"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/twitchtv/twirp"
	"net/http"
	"strconv"
)

func main() {
	cfg, err := config.ReadConfig("config.json")
	if err != nil {
		log.Fatal().Err(err).Msg("Config could not be read")
		return
	}

	http.DefaultClient.Transport = &util.LoggingRoundTripper{}

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

	trackerGgScraper := trackerggscraper.NewTrackerGgScraperProtobufClient(cfg.Services.TrackerGgScraper, http.DefaultClient)
	twitchAPI := twitch.NewAPI(cfg, cacheDB)

	botInstance := bot.NewBot(mainDB, cacheDB, cfg, twitchAPI, trackerGgScraper)

	serverInstance := server.NewServer(cfg, twitchAPI, mainDB, botInstance)
	err = serverInstance.Start(newRootContext())
	if err != nil {
		log.Fatal().Err(err).Msg("Could not start server instance!")
		return
	}

	select {}
}

func newRootContext() context.Context {
	ctx := context.Background()

	traceId := uuid.New().String()
	spanId := uuid.New().String()
	ctx = context.WithValue(ctx, "trace-id", traceId)
	ctx = context.WithValue(ctx, "span-id", spanId)

	ctxLogger := log.With().Str("trace-id", traceId).Str("span-id", spanId).Logger()
	ctx = ctxLogger.WithContext(ctx)

	outgoingHeaders := make(http.Header)
	outgoingHeaders.Set("trace-id", traceId)
	outgoingHeaders.Set("span-id", spanId)
	ctx, err := twirp.WithHTTPRequestHeaders(ctx, outgoingHeaders)
	if err != nil {
		ctxLogger.Panic().Err(err)
	}

	return ctx
}
