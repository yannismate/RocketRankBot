package server

import (
	"RocketRankBot/services/commander/internal/db"
	"RocketRankBot/services/commander/internal/twitch"
	"RocketRankBot/services/commander/internal/util"
	"context"
	"fmt"
	"github.com/rs/zerolog/log"
	"io"
	"net/http"
	"slices"
	"strings"
)

const authStateCookieName = "twitch_oauth_state"

var userScopes = []string{"channel:bot"}
var botScopes = []string{"user:bot", "user:write:chat", "user:read:chat"}

func (s *server) handleAuth(w http.ResponseWriter, r *http.Request) {
	state := util.RandomAlphanumericalString(32)

	authUrl := s.twitch.GenerateAuthorizeURL(userScopes, state)

	cookie := http.Cookie{
		Name:     authStateCookieName,
		Value:    state,
		MaxAge:   1800,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}

	http.SetCookie(w, &cookie)
	http.Redirect(w, r, authUrl.String(), http.StatusSeeOther)
}

func (s *server) handleAuthBot(w http.ResponseWriter, r *http.Request) {
	state := util.RandomAlphanumericalString(32)

	authUrl := s.twitch.GenerateAuthorizeURL(botScopes, state)

	cookie := http.Cookie{
		Name:     authStateCookieName,
		Value:    state,
		MaxAge:   1800,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}

	http.SetCookie(w, &cookie)
	http.Redirect(w, r, authUrl.String(), http.StatusSeeOther)
}

func (s *server) handleAuthCallback(w http.ResponseWriter, r *http.Request) {
	stateCookie, err := r.Cookie(authStateCookieName)

	if err != nil || stateCookie.Valid() != nil || len(stateCookie.Value) == 0 || r.URL.Query().Get("state") != stateCookie.Value {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, "Invalid auth state, please try again.")
		return
	}
	if r.URL.Query().Has("error") {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, "Authorization denied by user.")
		return
	}

	tokenResponse, err := s.twitch.GetTokenWithCode(r.Context(), r.URL.Query().Get("code"))
	if err != nil {
		return
	}

	user, err := s.twitch.GetOwnUser(r.Context(), tokenResponse.AccessToken)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = io.WriteString(w, fmt.Sprint("Error getting user info from Twitch. trace-id: ", r.Context().Value("trace-id")))
		return
	}

	log.Ctx(r.Context()).Info().Str("user_id", user.Data[0].ID).Str("user_login", user.Data[0].Login).Msg("Successfully created token for user")

	if strings.ToLower(user.Data[0].Login) != strings.ToLower(s.botTwitchUserName) {
		for _, s := range userScopes {
			if !slices.Contains(tokenResponse.Scope, s) {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = io.WriteString(w, "Authorization is missing required scope: "+s)
				return
			}
		}
	}

	ctx := context.WithoutCancel(r.Context())

	_, userExists, err := s.db.FindUser(ctx, user.Data[0].ID)
	if err != nil {
		_, _ = io.WriteString(w, fmt.Sprint("Error saving user data. Please try again later. trace-id: ", ctx.Value("trace-id")))
		log.Ctx(ctx).Error().Err(err).Msg("Error adding EventSub subscription to database")
		return
	}

	if userExists {
		// Delete old Subscriptions
		oldSubs, err := s.db.FindEventSubSubscriptionsForTwitchUserID(ctx, user.Data[0].ID)
		if err == nil {
			for _, oldSub := range *oldSubs {
				_ = s.twitch.DeleteEventSubSubscription(ctx, oldSub.SubscriptionID)
			}
		}
	}

	transport, err := s.twitch.EventSubTransport(ctx)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = io.WriteString(w, fmt.Sprint("Internal server error. Please try again later. trace-id: ", ctx.Value("trace-id")))
		return
	}

	subReq := twitch.CreateEventSubSubscriptionRequest{
		Type:      twitch.EventSubTypeChatMessage,
		Version:   twitch.EventSubVersionChatMessage,
		Condition: s.twitch.BotUserCondition(user.Data[0].ID),
		Transport: *transport,
	}

	subscriptionID, err := s.twitch.CreateEventSubSubscription(ctx, subReq)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = io.WriteString(w, fmt.Sprint("Error creating Twitch EventSub subscription. Please try again later. trace-id: ", ctx.Value("trace-id")))
		return
	}

	dbSub := db.EventSubSubscription{
		SubscriptionID: *subscriptionID,
		TwitchUserID:   user.Data[0].ID,
		Topic:          twitch.EventSubTypeChatMessage,
	}
	err = s.db.AddEventSubSubscription(ctx, &dbSub)
	if err != nil {
		_, _ = io.WriteString(w, fmt.Sprint("Error creating saving EventSub subscription. Please try again later. trace-id: ", ctx.Value("trace-id")))
		log.Ctx(ctx).Error().Err(err).Msg("Error adding EventSub subscription to database")
		return
	}

	if !userExists {
		err := s.db.AddUser(ctx, &db.BotUser{
			TwitchUserID:    user.Data[0].ID,
			IsAuthenticated: true,
		})
		if err != nil {
			_, _ = io.WriteString(w, fmt.Sprint("Error saving user data. Please try again later. trace-id: ", ctx.Value("trace-id")))
			log.Ctx(ctx).Error().Err(err).Msg("Error adding EventSub subscription to database")
			return
		}
	} else {
		err := s.db.UpdateUserAuthenticationFlag(ctx, user.Data[0].ID, true)
		if err != nil {
			_, _ = io.WriteString(w, fmt.Sprint("Error saving user data. Please try again later. trace-id: ", ctx.Value("trace-id")))
			log.Ctx(ctx).Error().Err(err).Msg("Error adding EventSub subscription to database")
			return
		}
	}

	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, "Authentication succeeded! You can close this tab and continue with the setup.")
	log.Ctx(r.Context()).Info().Str("user_id", user.Data[0].ID).Str("user_login", user.Data[0].Login).Msg("User registration completed")

}
