package server

import (
	"RocketRankBot/services/commander/internal/bot"
	"RocketRankBot/services/commander/internal/config"
	"RocketRankBot/services/commander/internal/db"
	"RocketRankBot/services/commander/internal/twitch"
	"context"
	"github.com/rs/zerolog/log"
	"net/http"
	"strconv"
)

type Server interface {
	Start(ctx context.Context) error
}

type server struct {
	bindAddress         string
	baseUrl             string
	twitch              twitch.API
	twitchWebhookSecret string
	db                  db.MainDB
	cache               db.CacheDB
	bot                 bot.Bot
	commandPrefix       string
	botTwitchUserName   string
	adminsUserIDs       []string
}

func NewServer(cfg *config.CommanderConfig, twitchAPI twitch.API, mainDB db.MainDB, cacheDB db.CacheDB, bot bot.Bot) Server {
	return &server{
		bindAddress:         ":" + strconv.Itoa(cfg.AppPort),
		baseUrl:             cfg.BaseURL,
		twitch:              twitchAPI,
		twitchWebhookSecret: cfg.Twitch.WebHookSecret,
		db:                  mainDB,
		cache:               cacheDB,
		bot:                 bot,
		commandPrefix:       cfg.CommandPrefix,
		botTwitchUserName:   cfg.Twitch.BotUserName,
		adminsUserIDs:       cfg.AdminUserIDs,
	}
}

func (s *server) Start(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/auth", s.handleAuth)
	mux.HandleFunc("/authbot", s.handleAuthBot)
	mux.HandleFunc("/callback", s.handleAuthCallback)
	mux.HandleFunc("/webhooks/twitch", s.handleTwitchWebHook)

	log.Ctx(ctx).Info().Str("bind_address", s.bindAddress).Msg("Starting HTTP server")

	go func() {
		err := http.ListenAndServe(s.bindAddress, WithLogging(false, mux))
		if err != nil {
			log.Ctx(ctx).Fatal().Err(err).Msg("HTTP server failed!")
		}
	}()
	err := s.twitch.CheckTransport(ctx)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("Twitch conduit is not working as expected")
		return err
	}

	return nil
}
