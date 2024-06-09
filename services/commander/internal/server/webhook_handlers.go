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
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/twitchtv/twirp"
	"io"
	"net/http"
	"strings"
	"time"
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
	if subtle.ConstantTimeCompare([]byte(signature), []byte(calculatedSignature)) != 1 {
		w.WriteHeader(http.StatusForbidden)
		log.Ctx(r.Context()).Warn().Str("calculated_sig", calculatedSignature).Str("received_sig", signature).Msg("received webhook call with invalid signature")
		return
	}

	parsedTimestamp, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Ctx(r.Context()).Warn().Err(err).Str("timestamp", timestamp).Msg("received webhook call with invalid timestamp")
		return
	}

	if parsedTimestamp.Before(time.Now().Add(-10 * time.Minute)) {
		w.WriteHeader(http.StatusForbidden)
		log.Ctx(r.Context()).Warn().Err(err).Time("timestamp", parsedTimestamp).Msg("received webhook call with outdated timestamp")
		return
	}

	switch messageType {
	case "webhook_callback_verification":
		s.handleWebHookChallenge(w, r, bodyData)
	case "notification":
		s.handleWebHookNotification(w, r, bodyData)
	case "revocation":
		s.handleWebHookRevocation(w, r, bodyData)
	default:
		w.WriteHeader(http.StatusBadRequest)
		log.Ctx(r.Context()).Warn().Err(err).Str("timestamp", timestamp).Msg("received validated webhook call with invalid message type")
		return
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

	log.Ctx(r.Context()).Info().Str("challenge", challenge.Challenge).Msg("Received webhook challenge")

	w.Header().Set("Content-Type", "text/plain")
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

	go s.bot.ExecutePossibleCommand(NewBotContext(r.Context()), &ipc)
}

func NewBotContext(parentContext context.Context) context.Context {
	ctx := context.Background()
	traceId := parentContext.Value("trace-id").(string)
	parentSpanId := parentContext.Value("span-id").(string)
	spanId := uuid.New().String()

	ctx = context.WithValue(ctx, "trace-id", traceId)
	ctx = context.WithValue(ctx, "parent-span-id", parentContext)
	ctx = context.WithValue(ctx, "span-id", spanId)

	ctxLogger := log.With().Str("trace-id", traceId).Str("parent-span-id", parentSpanId).Str("span-id", spanId).Logger()
	ctx = ctxLogger.WithContext(ctx)

	outgoingHeaders := make(http.Header)
	outgoingHeaders.Set("trace-id", traceId)
	outgoingHeaders.Set("span-id", spanId)
	ctx, err := twirp.WithHTTPRequestHeaders(ctx, outgoingHeaders)
	if err != nil {
		ctxLogger.Panic().Err(err)
	}
	return ctx
}

func (s *server) calculateHMAC(messageID string, timestamp string, requestBody string) string {
	mac := hmac.New(sha256.New, []byte(s.twitchWebhookSecret))
	mac.Write([]byte((messageID + timestamp + requestBody)))
	result := mac.Sum(nil)
	return hex.EncodeToString(result)
}
