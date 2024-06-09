package twitch

import (
	"RocketRankBot/services/commander/internal/config"
	"RocketRankBot/services/commander/internal/db"
	"context"
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
}

func NewAPI(cfg *config.CommanderConfig, cache db.CacheDB) API {
	return &api{
		clientID:      cfg.Twitch.ClientID,
		clientSecret:  cfg.Twitch.ClientSecret,
		botUserID:     cfg.Twitch.BotUserID,
		redirectURI:   cfg.BaseURL + "/callback",
		webHookURL:    cfg.BaseURL + "/webhooks/twitch",
		webHookSecret: cfg.Twitch.WebHookSecret,
		botConduitID:  "",
		cache:         cache,
	}
}
