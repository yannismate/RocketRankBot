package config

import (
	"encoding/json"
	"io"
	"os"
)

type CommanderConfig struct {
	AppPort   int
	AdminPort int

	DB struct {
		Main  string
		Cache string
	}

	Services struct {
		TwitchConnector  string
		TrackerGgScraper string
	}

	TTL struct {
		Commands int
		Ranks    int
	}
	CommandTimeoutSeconds int
	BotChannelName        string
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

	return &cfg, nil
}
