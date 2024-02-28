package config

import (
	"encoding/json"
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

	return &cfg, nil
}
