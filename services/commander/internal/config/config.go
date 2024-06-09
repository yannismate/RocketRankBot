package config

import (
	"encoding/json"
	"github.com/rs/zerolog/log"
	"io"
	"os"
)

type CommanderConfig struct {
	AppPort   int
	AdminPort int
	BaseURL   string

	DB struct {
		Main  string
		Cache string
	}

	Services struct {
		TrackerGgScraper string
	}

	TTL struct {
		Commands int
		Ranks    int
	}

	Twitch struct {
		ClientID      string
		ClientSecret  string
		BotUserID     string
		BotUserName   string
		WebHookSecret string
	}

	CommandPrefix         string
	CommandTimeoutSeconds int
}

func ReadConfig(path string) (*CommanderConfig, error) {
	cfgFile, err := os.Open(path)

	if err != nil {
		return nil, err
	}

	defer cfgFile.Close()

	fileBytes, err := io.ReadAll(cfgFile)
	if err != nil {
		return nil, err
	}

	var cfg CommanderConfig
	err = json.Unmarshal(fileBytes, &cfg)
	if err != nil {
		return nil, err
	}

	cfg.Twitch.ClientSecret = os.Getenv("TWITCH_CLIENT_SECRET")
	if len(cfg.Twitch.ClientSecret) == 0 {
		log.Fatal().Msg("Twitch Client Secret is empty.")
	}

	cfg.Twitch.WebHookSecret = os.Getenv("TWITCH_WEBHOOK_SECRET")
	if len(cfg.Twitch.WebHookSecret) == 0 {
		log.Fatal().Msg("Twitch WebHook Secret is empty.")
	}

	return &cfg, nil
}
