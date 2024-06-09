package twitch

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/rs/zerolog/log"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const twitchAuthorizeURL = "https://id.twitch.tv/oauth2/authorize"
const twitchTokenURL = "https://id.twitch.tv/oauth2/authorize"
const twitchValidateURL = "https://id.twitch.tv/oauth2/validate"

var (
	ErrTokenRequestFailed = errors.New("token request failed with non-200 status code")
)

func (api *api) GenerateAuthorizeURL(scopes []string, state string) *url.URL {
	authUrl, _ := url.Parse(twitchAuthorizeURL)

	params := url.Values{}
	params.Add("client_id", api.clientID)
	params.Add("redirect_uri", api.redirectURI)
	params.Add("response_type", "code")
	params.Add("scope", strings.Join(scopes, "+"))
	params.Add("state", state)

	authUrl.RawQuery = params.Encode()
	return authUrl
}

type TokenResponse struct {
	AccessToken  string   `json:"access_token"`
	ExpiresIn    int      `json:"expires_in"`
	RefreshToken string   `json:"refresh_token"`
	Scope        []string `json:"scope"`
	TokenType    string   `json:"token_type"`
}

func (api *api) GetTokenWithCode(ctx context.Context, code string) (*TokenResponse, error) {
	params := url.Values{}
	params.Add("client_id", api.clientID)
	params.Add("client_secret", api.clientSecret)
	params.Add("code", code)
	params.Add("grant_type", "authorization_code")
	params.Add("redirect_uri", api.redirectURI)

	return doTokenRequest(ctx, params)
}

func (api *api) GetTokenWithRefreshToken(ctx context.Context, refreshToken string) (*TokenResponse, error) {
	params := url.Values{}
	params.Add("client_id", api.clientID)
	params.Add("client_secret", api.clientSecret)
	params.Add("grant_type", "refresh_token")
	params.Add("refresh_token", refreshToken)

	return doTokenRequest(ctx, params)
}

func (api *api) getAppToken(ctx context.Context) (*string, error) {
	appState, cacheHit, err := api.cache.GetCachedAppState(ctx)
	if err != nil && cacheHit && len(appState.TwitchAppToken) != 0 && appState.TwitchAppTokenExpiry.After(time.Now()) {
		return &appState.TwitchAppToken, nil
	}

	log.Ctx(ctx).Info().Msg("Fetching new Twitch app token")
	params := url.Values{}
	params.Add("client_id", api.clientID)
	params.Add("client_secret", api.clientSecret)
	params.Add("grant_type", "client_credentials")

	res, err := doTokenRequest(ctx, params)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("Could not fetch Twitch app token!")
		return nil, err
	}

	log.Ctx(ctx).Info().Msg("New app token acquired, updating cache")
	appState.TwitchAppToken = res.AccessToken
	appState.TwitchAppTokenExpiry = time.Now().Add(time.Second * time.Duration(res.ExpiresIn))
	_ = api.cache.SetCachedAppState(ctx, appState)

	log.Ctx(ctx).Info().Msg("App token written to cache.")
	return &res.AccessToken, nil
}

func doTokenRequest(ctx context.Context, params url.Values) (*TokenResponse, error) {
	req, err := http.NewRequest("POST", twitchTokenURL, bytes.NewBufferString(params.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = req.WithContext(ctx)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return nil, ErrTokenRequestFailed
	}

	resData, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	tokenResponse := TokenResponse{}
	err = json.Unmarshal(resData, &tokenResponse)
	if err != nil {
		return nil, err
	}

	return &tokenResponse, nil
}

type ValidateTokenResponse struct {
	Login  string   `json:"login"`
	Scopes []string `json:"scopes"`
	UserID string   `json:"user_id"`
}
