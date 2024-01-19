package main

import (
	"RocketRankBot/services/commander/internal/bot"
	"RocketRankBot/services/commander/internal/config"
	"RocketRankBot/services/commander/internal/db"
	"RocketRankBot/services/commander/internal/server"
	"RocketRankBot/services/commander/rpc/commander"
	"RocketRankBot/services/commander/rpc/trackerggscraper"
	"RocketRankBot/services/commander/rpc/twitchconnector"
	"go.uber.org/zap"
	"net/http"
	"strconv"
)

func main() {
	// TODO: configure logging and metrics

	cfg, err := config.ReadConfig("config.json")
	if err != nil {
		zap.L().Fatal("Config could not be read", zap.Error(err))
		return
	}

	mainDB, err := db.NewMainDB(cfg)
	if err != nil {
		zap.L().Fatal("Could not connect to main database", zap.Error(err))
		return
	}

	cacheDB, err := db.NewCache(cfg)
	if err != nil {
		zap.L().Fatal("Could not connect to cache database", zap.Error(err))
		return
	}

	twitchConnector := twitchconnector.NewTwitchConnectorProtobufClient(cfg.Services.TwitchConnector, http.DefaultClient)
	trackerGgScraper := trackerggscraper.NewTrackerGgScraperProtobufClient(cfg.Services.TrackerGgScraper, http.DefaultClient)

	botInstance := bot.NewBot(mainDB, cacheDB, cfg, twitchConnector, trackerGgScraper)

	serverInstance := server.NewServer(botInstance)
	twirpHandler := commander.NewCommanderServer(&serverInstance)

	err = http.ListenAndServe(":"+strconv.Itoa(cfg.AppPort), twirpHandler)
	if err != nil {
		zap.L().Fatal("HTTP Listener error", zap.Error(err))
		return
	}
}
