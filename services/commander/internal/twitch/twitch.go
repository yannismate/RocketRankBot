package twitch

import (
	"RocketRankBot/services/commander/internal/config"
	"RocketRankBot/services/commander/internal/db"
	"RocketRankBot/services/commander/internal/util"
	"context"
	"github.com/rs/zerolog/log"
	"net/http"
	"net/url"
)

type API interface {
	GenerateAuthorizeURL(scopes []string, state string) *url.URL
	GetTokenWithCode(ctx context.Context, code string) (*TokenResponse, error)
	GetTokenWithRefreshToken(ctx context.Context, refreshToken string) (*TokenResponse, error)
	CreateEventSubSubscription(ctx context.Context, createSubReq CreateEventSubSubscriptionRequest) (*string, error)
	DeleteEventSubSubscription(ctx context.Context, subscriptionID string) error
	GetOwnUser(ctx context.Context, userToken string) (*UserResponse, error)
	BotUserCondition(broadcasterID string) BroadcasterAndUserCondition
	EventSubTransport(ctx context.Context) (*EventSubTransportReq, error)
	SendChatMessage(ctx context.Context, broadcasterID string, message string, replyMessageID *string) error
	CheckTransport(ctx context.Context) error
}

type api struct {
	clientID      string
	clientSecret  string
	botUserID     string
	redirectURI   string
	webHookURL    string
	webHookSecret string
	botConduitID  string
	cache         db.CacheDB
	httpClient    *http.Client
}

func NewAPI(cfg *config.CommanderConfig, cache db.CacheDB) API {
	httpClient := &http.Client{
		Transport: &util.LoggingRoundTripper{},
	}

	return &api{
		clientID:      cfg.Twitch.ClientID,
		clientSecret:  cfg.Twitch.ClientSecret,
		botUserID:     cfg.Twitch.BotUserID,
		redirectURI:   cfg.BaseURL + "/callback",
		webHookURL:    cfg.BaseURL + "/webhooks/twitch",
		webHookSecret: cfg.Twitch.WebHookSecret,
		botConduitID:  "",
		cache:         cache,
		httpClient:    httpClient,
	}
}

func (api *api) CheckTransport(ctx context.Context) error {
	id, err := api.getBotConduitID(ctx)
	if err != nil {
		return err
	}
	log.Ctx(ctx).Info().Str("conduit_id", *id).Msg("Twitch conduit is set up.")
	return nil
}
