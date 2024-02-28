package main

import (
	"RocketRankBot/services/twitchconnector/internal/config"
	"RocketRankBot/services/twitchconnector/internal/connector"
	"RocketRankBot/services/twitchconnector/internal/server"
	"RocketRankBot/services/twitchconnector/rpc/commander"
	"RocketRankBot/services/twitchconnector/rpc/twitchconnector"
	"github.com/rs/zerolog/log"
	"net/http"
	"strconv"
)

func main() {
	//TODO: metrics

	cfg, err := config.ReadConfig("config.json")
	if err != nil {
		log.Fatal().Err(err).Msg("Config could not be read")
		return
	}

	commanderClient := commander.NewCommanderProtobufClient(cfg.Services.Commander, http.DefaultClient)

	connectorInstance := connector.NewConnector(cfg, commanderClient)
	connectorInstance.Start()

	serverInstance := server.NewServer(connectorInstance)
	twirpHandler := server.WithLogging(twitchconnector.NewTwitchConnectorServer(&serverInstance))

	err = http.ListenAndServe(":"+strconv.Itoa(cfg.AppPort), twirpHandler)
	if err != nil {
		log.Fatal().Err(err).Msg("HTTP Listener error")
		return
	}
}
