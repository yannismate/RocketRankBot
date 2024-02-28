package config

import (
	"encoding/json"
	"github.com/rs/zerolog/log"
	"io"
	"os"
)

type TwitchconnectorConfig struct {
	AppPort   int
	AdminPort int

	Services struct {
		Commander string
	}

	Twitch struct {
		Login        string
		ClientID     string
		ClientSecret string
	}

	CommandPrefix string
}

func ReadConfig(path string) (*TwitchconnectorConfig, error) {
	cfgFile, err := os.Open(path)

	if err != nil {
		return nil, err
	}

	defer cfgFile.Close()

	fileBytes, err := io.ReadAll(cfgFile)
	if err != nil {
		return nil, err
	}

	var cfg TwitchconnectorConfig
	err = json.Unmarshal(fileBytes, &cfg)
	if err != nil {
		return nil, err
	}

	cfg.Twitch.ClientID = os.Getenv("TWITCH_CLIENT_ID")
	cfg.Twitch.ClientSecret = os.Getenv("TWITCH_CLIENT_SECRET")

	if len(cfg.Twitch.ClientID) == 0 || len(cfg.Twitch.ClientSecret) == 0 {
		log.Warn().Msg("Twitch Client ID or Secret are empty.")
	}

	return &cfg, nil
}
