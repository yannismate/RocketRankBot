package main

import (
	"RocketRankBot/services/commander/internal/bot"
	"RocketRankBot/services/commander/internal/config"
	"RocketRankBot/services/commander/internal/db"
	"RocketRankBot/services/commander/internal/server"
	"RocketRankBot/services/commander/rpc/commander"
	"RocketRankBot/services/commander/rpc/trackerggscraper"
	"RocketRankBot/services/commander/rpc/twitchconnector"
	"github.com/rs/zerolog/log"
	"net/http"
	"strconv"
)

func main() {
	// TODO: Metrics

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

	twitchConnector := twitchconnector.NewTwitchConnectorProtobufClient(cfg.Services.TwitchConnector, http.DefaultClient)
	trackerGgScraper := trackerggscraper.NewTrackerGgScraperProtobufClient(cfg.Services.TrackerGgScraper, http.DefaultClient)

	botInstance := bot.NewBot(mainDB, cacheDB, cfg, twitchConnector, trackerGgScraper)

	serverInstance := server.NewServer(botInstance)
	twirpHandler := server.WithLogging(commander.NewCommanderServer(&serverInstance))

	err = http.ListenAndServe(":"+strconv.Itoa(cfg.AppPort), twirpHandler)
	if err != nil {
		log.Fatal().Err(err).Msg("HTTP Listener error")
		return
	}
}
