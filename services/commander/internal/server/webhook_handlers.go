package server

import (
	"RocketRankBot/services/commander/internal/bot"
	"RocketRankBot/services/commander/internal/twitch"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"github.com/rs/zerolog/log"
	"io"
	"net/http"
	"strconv"
	"strings"
)

const (
	headerEventSubMessageID        = "Twitch-Eventsub-Message-Id"
	headerEventSubMessageType      = "Twitch-Eventsub-Message-Type"
	headerEventSubSignature        = "Twitch-Eventsub-Message-Signature"
	headerEventSubTimestamp        = "Twitch-Eventsub-Message-Timestamp"
	headerEventSubSubscriptionType = "Twitch-Eventsub-Subscription-Type"
)

type eventSubChallengeRequest struct {
	Challenge string `json:"challenge"`
}

func (s *server) handleTwitchWebHook(w http.ResponseWriter, r *http.Request) {
	messageID := r.Header.Get(headerEventSubMessageID)
	messageType := r.Header.Get(headerEventSubMessageType)
	signature := r.Header.Get(headerEventSubSignature)
	timestamp := r.Header.Get(headerEventSubTimestamp)

	if len(signature) == 0 || len(timestamp) == 0 {
		w.WriteHeader(http.StatusForbidden)
		log.Ctx(r.Context()).Warn().Msg("received webhook call without signature or timestamp")
		return
	}

	bodyData, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	calculatedSignature := "sha256=" + s.calculateHMAC(messageID, timestamp, string(bodyData))
	if subtle.ConstantTimeCompare([]byte(signature), []byte(calculatedSignature)) != 0 {
		w.WriteHeader(http.StatusForbidden)
		log.Ctx(r.Context()).Warn().Msg("received webhook call with invalid signature")
		return
	}

	switch messageType {
	case "challenge":
		s.handleWebHookChallenge(w, r, bodyData)
	case "notification":
		s.handleWebHookNotification(w, r, bodyData)
	case "revocation":
		s.handleWebHookRevocation(w, r, bodyData)
	}
}

func (s *server) handleWebHookChallenge(w http.ResponseWriter, r *http.Request, bodyData []byte) {
	challenge := eventSubChallengeRequest{}
	err := json.Unmarshal(bodyData, &challenge)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Ctx(r.Context()).Warn().Err(err).Msg("could not parse received webhook challenge")
		return
	}

	w.Header().Set("Content-Type", strconv.Itoa(len(challenge.Challenge)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(challenge.Challenge))
}

type webHookRevocation struct {
	Subscription struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	} `json:"subscription"`
}

func (s *server) handleWebHookRevocation(w http.ResponseWriter, r *http.Request, bodyData []byte) {
	revocation := webHookRevocation{}
	err := json.Unmarshal(bodyData, &revocation)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Ctx(r.Context()).Warn().Err(err).Msg("could not parse received webhook revocation")
		return
	}

	w.WriteHeader(http.StatusNoContent)

	ctx := context.WithoutCancel(r.Context())

	sub, hasSub, err := s.db.FindEventSubSubscriptionByID(ctx, revocation.Subscription.ID)
	if err != nil {
		log.Ctx(r.Context()).Error().Err(err).Msg("could not fetch eventsub subcription for deletion")
		return
	}
	if !hasSub {
		return
	}

	err = s.db.UpdateUserAuthenticationFlag(ctx, sub.TwitchUserID, false)
	if err != nil {
		log.Ctx(r.Context()).Error().Err(err).Msg("could not update user flag after eventsub subcription deletion")
		return
	}

}

type webHookNotificationChat struct {
	Event struct {
		BroadcasterUserID    string `json:"broadcaster_user_id"`
		BroadcasterUserLogin string `json:"broadcaster_user_login"`
		ChatterUserID        string `json:"chatter_user_id"`
		ChatterUserLogin     string `json:"chatter_user_login"`
		MessageID            string `json:"message_id"`
		Message              struct {
			Text string `json:"text"`
		} `json:"message"`
		Badges []struct {
			SetID string `json:"set_id"`
		} `json:"badges"`
	} `json:"event"`
}

func (s *server) handleWebHookNotification(w http.ResponseWriter, r *http.Request, bodyData []byte) {
	subscriptionType := r.Header.Get(headerEventSubSubscriptionType)
	if subscriptionType != twitch.EventSubTypeChatMessage {
		// ignore non-chat subscriptions for now
		w.WriteHeader(http.StatusNoContent)
		return
	}
	notificationChat := webHookNotificationChat{}

	err := json.Unmarshal(bodyData, &notificationChat)
	if err != nil {
		log.Ctx(r.Context()).Warn().Err(err).Msg("could not parse webhook chat notification")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	command := notificationChat.Event.Message.Text
	if strings.HasPrefix(strings.ToLower(command), "@"+strings.ToLower(s.botTwitchUserName)+" ") {
		command = command[(len(s.botTwitchUserName) + 2):]
	}

	if !strings.HasPrefix(command, s.commandPrefix) {
		return
	}

	isMod := false
	isBroadcaster := false

	for _, badge := range notificationChat.Event.Badges {
		if badge.SetID == "moderator" {
			isMod = true
		}
		if badge.SetID == "broadcaster" {
			isBroadcaster = true
		}
	}

	ipc := bot.IncomingPossibleCommand{
		Command:       command,
		IsModerator:   isMod,
		IsBroadcaster: isBroadcaster,
		ChannelID:     notificationChat.Event.BroadcasterUserID,
		ChannelLogin:  notificationChat.Event.BroadcasterUserLogin,
		SenderID:      notificationChat.Event.ChatterUserID,
		SenderLogin:   notificationChat.Event.ChatterUserLogin,
		MessageID:     notificationChat.Event.MessageID,
	}

	go s.bot.ExecutePossibleCommand(context.WithoutCancel(r.Context()), &ipc)
}

func (s *server) calculateHMAC(messageID string, timestamp string, requestBody string) string {
	mac := hmac.New(sha256.New, []byte(s.twitchWebhookSecret))
	mac.Write([]byte((messageID + timestamp + requestBody)))
	result := mac.Sum(nil)
	return hex.EncodeToString(result)
}