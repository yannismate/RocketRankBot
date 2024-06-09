package twitch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

const (
	twitchEventSubURL          = "https://api.twitch.tv/helix/eventsub/subscriptions"
	EventSubTypeChatMessage    = "channel.chat.message"
	EventSubVersionChatMessage = "1"
)

type CreateEventSubSubscriptionRequest struct {
	Type      string               `json:"type"`
	Version   string               `json:"version"`
	Condition interface{}          `json:"condition"`
	Transport EventSubTransportReq `json:"shards"`
}
type EventSubTransportReq struct {
	Method    string `json:"method"`
	ConduitID string `json:"conduit_id"`
}
type createEventSubSubscriptionResponse struct {
	Data []struct {
		ID string `json:"id"`
	} `json:"data"`
}
type BroadcasterAndUserCondition struct {
	BroadcasterUserId string `json:"broadcaster_user_id"`
	UserId            string `json:"user_id"`
}

func (api *api) BotUserCondition(broadcasterID string) BroadcasterAndUserCondition {
	return BroadcasterAndUserCondition{BroadcasterUserId: broadcasterID, UserId: api.botUserID}
}

func (api *api) EventSubTransport(ctx context.Context) (*EventSubTransportReq, error) {
	conduitID, err := api.getBotConduitID(ctx)
	if err != nil {
		return nil, err
	}

	return &EventSubTransportReq{
		Method:    "conduit",
		ConduitID: *conduitID,
	}, nil
}

func (api *api) CreateEventSubSubscription(ctx context.Context, createSubReq CreateEventSubSubscriptionRequest) (*string, error) {
	appToken, err := api.getAppToken(ctx)
	if err != nil {
		return nil, err
	}

	bodyData, err := json.Marshal(createSubReq)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", twitchEventSubURL, bytes.NewReader(bodyData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Client-Id", api.clientID)
	req.Header.Set("Authorization", "Bearer "+*appToken)
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(ctx)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("create event sub subscription request failed with status code %d", res.StatusCode)
	}

	resData, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	createEventSubRes := createEventSubSubscriptionResponse{}
	err = json.Unmarshal(resData, &createEventSubRes)
	if err != nil {
		return nil, err
	}

	if len(createEventSubRes.Data) != 1 {
		return nil, fmt.Errorf("unexpected amount of event sub subscriptions received: %d", len(createEventSubRes.Data))
	}

	return &createEventSubRes.Data[0].ID, nil
}

func (api *api) DeleteEventSubSubscription(ctx context.Context, subscriptionID string) error {
	appToken, err := api.getAppToken(ctx)
	if err != nil {
		return err
	}

	reqUrl, _ := url.Parse(twitchEventSubURL)
	params := url.Values{}
	params.Set("id", subscriptionID)
	reqUrl.RawQuery = params.Encode()

	req, err := http.NewRequest("DELETE", reqUrl.String(), nil)
	if err != nil {
		return err
	}

	req.Header.Set("Client-Id", api.clientID)
	req.Header.Set("Authorization", "Bearer "+*appToken)
	req = req.WithContext(ctx)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusNoContent {
		return fmt.Errorf("delete event sub subscription request failed with status code %d", res.StatusCode)
	}

	return nil
}
