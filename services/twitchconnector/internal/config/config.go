package config

import (
	"encoding/json"
	"github.com/rs/zerolog/log"
	"io"
	"os"
	"strings"
)

type TwitchconnectorConfig struct {
	AppPort   int
	AdminPort int

	Services struct {
		Commander string
	}

	Twitch struct {
		Login string
		Token string
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

	cfg.Twitch.Token = os.Getenv("TWITCH_TOKEN")

	if len(cfg.Twitch.Token) == 0 {
		log.Fatal().Msg("Twitch Token is empty.")
	}
	if !strings.HasPrefix(cfg.Twitch.Token, "oauth:") {
		log.Warn().Msg("Twitch Token should have oauth prefix!")
		cfg.Twitch.Token = "oauth:" + cfg.Twitch.Token
	}

	return &cfg, nil
}
