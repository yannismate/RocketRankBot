package twitch

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
)

const twitchChatMessageURL = "https://api.twitch.tv/helix/chat/messages"

var (
	ErrBotUserNotAuthenticated = errors.New("bot user not authenticated")
)

type chatMessageSendRequest struct {
	BroadcasterID        string  `json:"broadcaster_id"`
	SenderID             string  `json:"sender_id"`
	Message              string  `json:"message"`
	ReplyParentMessageID *string `json:"reply_parent_message_id,omitempty"`
}

func (api *api) SendChatMessage(ctx context.Context, broadcasterID string, message string, replyMessageID *string) error {
	appToken, err := api.getAppToken(ctx)
	if err != nil {
		return err
	}

	bodyData, err := json.Marshal(chatMessageSendRequest{
		BroadcasterID:        broadcasterID,
		SenderID:             api.botUserID,
		Message:              message,
		ReplyParentMessageID: replyMessageID,
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", twitchChatMessageURL, bytes.NewReader(bodyData))
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

	if res.StatusCode == http.StatusUnauthorized {
		return ErrBotUserNotAuthenticated
	}

	return nil
}
