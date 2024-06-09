package main

import (
	"RocketRankBot/services/commander/internal/bot"
	"RocketRankBot/services/commander/internal/config"
	"RocketRankBot/services/commander/internal/db"
	"RocketRankBot/services/commander/internal/metrics"
	"RocketRankBot/services/commander/internal/server"
	"RocketRankBot/services/commander/internal/twitch"
	"RocketRankBot/services/commander/rpc/trackerggscraper"
	"context"
	"github.com/rs/zerolog/log"
	"net/http"
	"strconv"
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

	trackerGgScraper := trackerggscraper.NewTrackerGgScraperProtobufClient(cfg.Services.TrackerGgScraper, http.DefaultClient)
	twitchAPI := twitch.NewAPI(cfg, cacheDB)

	botInstance := bot.NewBot(mainDB, cacheDB, cfg, twitchAPI, trackerGgScraper)

	serverInstance := server.NewServer(cfg, twitchAPI, mainDB, botInstance)
	err = serverInstance.Start(context.Background())
	if err != nil {
		log.Fatal().Err(err).Msg("HTTP Server crashed")
		return
	}
}
