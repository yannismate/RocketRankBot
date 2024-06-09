package twitch

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const twitchUserURL = "https://api.twitch.tv/helix/users"

type UserResponse struct {
	Data []struct {
		ID    string `json:"id"`
		Login string `json:"login"`
	} `json:"data"`
}

func (api *api) GetOwnUser(ctx context.Context, userToken string) (*UserResponse, error) {
	req, err := http.NewRequest("GET", twitchUserURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Client-Id", api.clientID)
	req.Header.Set("Authorization", "Bearer "+userToken)
	req = req.WithContext(ctx)

	res, err := api.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get user request failed with status code %d", res.StatusCode)
	}

	resData, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	userRes := UserResponse{}
	err = json.Unmarshal(resData, &userRes)
	if err != nil {
		return nil, err
	}

	if len(userRes.Data) != 1 {
		return nil, fmt.Errorf("unexpected amount of users received: %d", len(userRes.Data))
	}

	return &userRes, nil
}
