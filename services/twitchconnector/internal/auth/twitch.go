package auth

import (
	"RocketRankBot/services/twitchconnector/internal/config"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	TwitchOAuthTokenURL = "https://id.twitch.tv/oauth2/token"
)

var (
	TwitchUnexpectedTokenTypeErr = errors.New("unexpected token type")
	TwitchEmptyTokenErr          = errors.New("token is empty")
)

type TwitchAuth interface {
	GetAccessToken() (*string, error)
	Invalidate()
}

type twitchAuth struct {
	clientID                 string
	clientSecret             string
	currentAccessToken       *string
	currentAccessTokenExpiry *time.Time
}

func NewTwitchAuth(cfg *config.TwitchconnectorConfig) TwitchAuth {
	return &twitchAuth{
		clientID:                 cfg.Twitch.ClientID,
		clientSecret:             cfg.Twitch.ClientSecret,
		currentAccessToken:       nil,
		currentAccessTokenExpiry: nil,
	}
}

type accessTokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	TokenType   string `json:"token_type"`
}

func (t *twitchAuth) GetAccessToken() (*string, error) {
	if t.currentAccessToken != nil && t.currentAccessTokenExpiry != nil && t.currentAccessTokenExpiry.After(time.Now()) {
		return t.currentAccessToken, nil
	}

	form := url.Values{}
	form.Add("client_id", t.clientID)
	form.Add("client_secret", t.clientSecret)
	form.Add("grant_type", "client_credentials")

	req, err := http.NewRequest("POST", TwitchOAuthTokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var tokenResponse accessTokenResponse
	err = json.NewDecoder(res.Body).Decode(&tokenResponse)
	if err != nil {
		return nil, err
	}

	if tokenResponse.TokenType != "bearer" {
		return nil, TwitchUnexpectedTokenTypeErr
	}
	if len(tokenResponse.AccessToken) == 0 {
		return nil, TwitchEmptyTokenErr
	}

	newExpiry := time.Now().Add(time.Second * time.Duration(tokenResponse.ExpiresIn))
	t.currentAccessTokenExpiry = &newExpiry
	t.currentAccessToken = &tokenResponse.AccessToken

	return t.currentAccessToken, nil
}

func (t *twitchAuth) Invalidate() {
	t.currentAccessTokenExpiry = nil
	t.currentAccessToken = nil
}
